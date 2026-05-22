package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"sub2api-go/internal/model"
)

const (
	KeyPrefixAccountBalance   = "account:balance:"
	KeyPrefixAccountRecharged = "account:recharged:"
	KeyPrefixAccountGrant     = "account:grant:"
	KeyPrefixKeySpent         = "key:spent:"
)

func accountRechargedKey(userID string) string {
	return KeyPrefixAccountRecharged + userID
}

func isPaidTopupType(txType string) bool {
	return txType == "topup" || txType == "admin_topup"
}

func accountBalanceKey(userID string) string {
	return KeyPrefixAccountBalance + userID
}

func accountGrantKey(userID, month string) string {
	return KeyPrefixAccountGrant + userID + ":" + month
}

func keySpentKey(keyID string) string {
	return KeyPrefixKeySpent + keyID
}

func currentGrantMonth() string {
	return time.Now().UTC().Format("2006-01")
}

func (s *RedisStore) GetAccountRechargedBalance(ctx context.Context, userID string) (float64, error) {
	key := accountRechargedKey(userID)
	balanceStr, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return s.bootstrapAccountRecharged(ctx, userID)
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(balanceStr, 64)
}

func (s *RedisStore) bootstrapAccountRecharged(ctx context.Context, userID string) (float64, error) {
	bal, _ := s.GetAccountBalance(ctx, userID)
	grantSum, _ := s.sumAccountTxAmountByType(ctx, userID, "monthly_grant")
	paidSum, _ := s.sumAccountTxAmountByType(ctx, userID, "topup", "admin_topup")
	recharged := bal - grantSum
	if recharged < 0 {
		recharged = 0
	}
	if paidSum > 0 && recharged > paidSum {
		recharged = paidSum
	}
	_ = s.client.Set(ctx, accountRechargedKey(userID), fmt.Sprintf("%.6f", recharged), 0).Err()
	return recharged, nil
}

func (s *RedisStore) sumAccountTxAmountByType(ctx context.Context, userID string, types ...string) (float64, error) {
	allowed := make(map[string]bool, len(types))
	for _, t := range types {
		allowed[t] = true
	}
	var sum float64
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 100).Result()
		if err != nil {
			break
		}
		for _, k := range keys {
			txJSON, err := s.client.Get(ctx, k).Result()
			if err != nil {
				continue
			}
			var tx model.Transaction
			if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
				continue
			}
			if tx.UserID != userID || !allowed[tx.Type] {
				continue
			}
			sum += tx.Amount
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return sum, nil
}

func (s *RedisStore) adjustAccountRecharged(ctx context.Context, userID string, consumeAmount float64, balanceBefore float64) error {
	if consumeAmount <= 0 {
		return nil
	}
	recharged, err := s.GetAccountRechargedBalance(ctx, userID)
	if err != nil {
		return err
	}
	bonus := balanceBefore - recharged
	if bonus < 0 {
		bonus = 0
	}
	fromBonus := consumeAmount
	if fromBonus > bonus {
		fromBonus = bonus
	}
	deductRecharged := consumeAmount - fromBonus
	if deductRecharged <= 0 {
		return nil
	}
	newR := recharged - deductRecharged
	if newR < 0 {
		newR = 0
	}
	return s.client.Set(ctx, accountRechargedKey(userID), fmt.Sprintf("%.6f", newR), 0).Err()
}

func (s *RedisStore) GetAccountBalance(ctx context.Context, userID string) (float64, error) {
	balanceStr, err := s.client.Get(ctx, accountBalanceKey(userID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(balanceStr, 64)
}

func (s *RedisStore) ensureAccountBalanceKey(ctx context.Context, userID string) error {
	key := accountBalanceKey(userID)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return s.client.Set(ctx, key, "0", 0).Err()
	}
	return nil
}

func (s *RedisStore) AccountTopup(ctx context.Context, userID string, amount float64, txType, note, stripePaymentID string, setHasPaid bool) error {
	if err := s.ensureAccountBalanceKey(ctx, userID); err != nil {
		return err
	}
	oldBalance, _ := s.GetAccountBalance(ctx, userID)
	newBalance, err := s.client.IncrByFloat(ctx, accountBalanceKey(userID), amount).Result()
	if err != nil {
		return err
	}
	if isPaidTopupType(txType) {
		_, _ = s.client.IncrByFloat(ctx, accountRechargedKey(userID), amount).Result()
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Balance = newBalance
	if setHasPaid && !user.HasPaid {
		user.HasPaid = true
		now := time.Now()
		user.FirstPaidAt = &now
	}
	user.UpdatedAt = time.Now()
	if err := s.UpdateUser(ctx, user); err != nil {
		return err
	}
	now := time.Now()
	actor := "system"
	if txType == "admin_topup" || txType == "admin_adjust" {
		actor = "admin"
	}
	if stripePaymentID != "" {
		actor = "stripe_webhook"
	}
	tx := model.Transaction{
		ID:              generateTxID(),
		UserID:          userID,
		Type:            txType,
		Amount:          amount,
		BalanceBefore:   oldBalance,
		BalanceAfter:    newBalance,
		StripePaymentID: stripePaymentID,
		Note:            note,
		Actor:           actor,
		CreatedAt:       now,
	}
	txJSON, _ := json.Marshal(tx)
	s.client.Set(ctx, KeyPrefixTransaction+tx.ID, txJSON, TransactionRedisTTL)
	s.writeThroughLedger(ctx, &tx)
	s.writeThroughUserAccount(ctx, user)
	return nil
}

func (s *RedisStore) AccountPreDeduct(ctx context.Context, userID string, amount float64) error {
	if err := s.ensureAccountBalanceKey(ctx, userID); err != nil {
		return err
	}
	balBefore, _ := s.GetAccountBalance(ctx, userID)
	result, err := s.scripts["pre_deduct"].Run(ctx, s.client,
		[]string{accountBalanceKey(userID)},
		fmt.Sprintf("%.6f", amount),
	).Int()
	if err != nil {
		return err
	}
	switch result {
	case 1:
		return s.adjustAccountRecharged(ctx, userID, amount, balBefore)
	case 0:
		return ErrInsufficientBalance
	default:
		return ErrUserNotFound
	}
}

func (s *RedisStore) AccountRefundPreDeduct(ctx context.Context, userID string, amount float64) error {
	_, err := s.scripts["refund"].Run(ctx, s.client,
		[]string{accountBalanceKey(userID)},
		fmt.Sprintf("%.6f", amount),
	).Float64()
	return err
}

func (s *RedisStore) AccountFinalizeDeduct(ctx context.Context, userID, keyID, txType, modelName, requestID string, preDeducted, actualAmount float64, usage model.Usage) error {
	balKey := accountBalanceKey(userID)
	oldBalance, _ := s.GetAccountBalance(ctx, userID)
	// pre_deduct already removed estimate; finalize adjusts diff
	_, err := s.scripts["finalize_deduct"].Run(ctx, s.client,
		[]string{balKey},
		fmt.Sprintf("%.6f", preDeducted),
		fmt.Sprintf("%.6f", actualAmount),
	).Float64()
	if err != nil {
		return err
	}
	newBalance, _ := s.GetAccountBalance(ctx, userID)

	if keyID != "" {
		_ = s.AddKeySpent(ctx, keyID, actualAmount)
	}

	user, uerr := s.GetUserByID(ctx, userID)
	if uerr == nil {
		user.Balance = newBalance
		user.UpdatedAt = time.Now()
		_ = s.UpdateUser(ctx, user)
		s.writeThroughUserAccount(ctx, user)
	}

	now := time.Now()
	tx := model.Transaction{
		ID:            generateTxID(),
		UserID:        userID,
		KeyID:         keyID,
		Type:          txType,
		Amount:        actualAmount,
		BalanceBefore: oldBalance,
		BalanceAfter:  newBalance,
		Model:         modelName,
		InputTokens:   usage.PromptTokens,
		OutputTokens:  usage.CompletionTokens,
		RequestID:     requestID,
		Actor:         "system",
		CreatedAt:     now,
	}
	txJSON, _ := json.Marshal(tx)
	s.client.Set(ctx, KeyPrefixTransaction+tx.ID, txJSON, TransactionRedisTTL)
	s.writeThroughLedger(ctx, &tx)

	if extra := actualAmount - preDeducted; extra > 0 {
		balBefore, _ := s.GetAccountBalance(ctx, userID)
		_ = s.adjustAccountRecharged(ctx, userID, extra, balBefore+extra)
	}
	return nil
}

func (s *RedisStore) TryMonthlyGrant(ctx context.Context, userID string, grantUSD float64) (bool, error) {
	if grantUSD <= 0 {
		return false, nil
	}
	month := currentGrantMonth()
	gkey := accountGrantKey(userID, month)
	ok, err := s.client.SetNX(ctx, gkey, "1", 32*24*time.Hour).Result()
	if err != nil || !ok {
		return false, err
	}
	if err := s.AccountTopup(ctx, userID, grantUSD, "monthly_grant", "monthly grant "+month, "", false); err != nil {
		_ = s.client.Del(ctx, gkey)
		return false, err
	}
	user, err := s.GetUserByID(ctx, userID)
	if err == nil {
		user.LastMonthlyGrantMonth = month
		user.UpdatedAt = time.Now()
		_ = s.UpdateUser(ctx, user)
		s.writeThroughUserAccount(ctx, user)
	}
	return true, nil
}

func (s *RedisStore) GetKeySpentTotal(ctx context.Context, keyID string) (float64, error) {
	v, err := s.client.Get(ctx, keySpentKey(keyID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

func (s *RedisStore) CheckKeySpendLimit(ctx context.Context, keyID string, spendLimit *float64, additionalAmount float64) error {
	if spendLimit == nil || *spendLimit <= 0 {
		return nil
	}
	spent, err := s.GetKeySpentTotal(ctx, keyID)
	if err != nil {
		return err
	}
	if spent+additionalAmount > *spendLimit+1e-9 {
		return ErrKeySpendLimitExceeded
	}
	return nil
}

func (s *RedisStore) AddKeySpent(ctx context.Context, keyID string, amount float64) error {
	if keyID == "" || amount <= 0 {
		return nil
	}
	return s.client.IncrByFloat(ctx, keySpentKey(keyID), amount).Err()
}

func (s *RedisStore) SetKeySpendLimit(ctx context.Context, keyHash string, spendLimit *float64) error {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return err
	}
	key.SpendLimit = spendLimit
	key.UpdatedAt = time.Now()
	keyJSON, _ := json.Marshal(key)
	return s.client.Set(ctx, KeyPrefixAPIKey+keyHash, keyJSON, 0).Err()
}

func (s *RedisStore) ListAccountTransactions(ctx context.Context, userID string, limit, offset int) ([]*model.Transaction, int, error) {
	// Scan tx:* and filter by user_id (acceptable for MVP; optimize with user index later)
	var cursor uint64
	var all []*model.Transaction
	for {
		keys, next, err := s.client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 200).Result()
		if err != nil {
			return nil, 0, err
		}
		for _, k := range keys {
			raw, err := s.client.Get(ctx, k).Result()
			if err != nil {
				continue
			}
			var tx model.Transaction
			if json.Unmarshal([]byte(raw), &tx) != nil {
				continue
			}
			if tx.UserID == userID {
				cp := tx
				all = append(all, &cp)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	// newest first
	sortTransactionsDesc(all)
	total := len(all)
	if offset >= total {
		return []*model.Transaction{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func sortTransactionsDesc(txs []*model.Transaction) {
	for i := 0; i < len(txs); i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[j].CreatedAt.After(txs[i].CreatedAt) {
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
	}
}
