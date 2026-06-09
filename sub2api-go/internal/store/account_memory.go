package store

import (
	"context"
	"time"

	"sub2api-go/internal/model"
)

func (s *MemoryStore) GetAccountBalance(ctx context.Context, userID string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usersById[userID]
	if !ok {
		return 0, ErrUserNotFound
	}
	return u.Balance, nil
}

func (s *MemoryStore) GetAccountRechargedBalance(ctx context.Context, userID string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usersById[userID]
	if !ok {
		return 0, ErrUserNotFound
	}
	return u.RechargedBalance, nil
}

func (s *MemoryStore) memoryAdjustRechargedLocked(u *model.User, consumeAmount float64, balanceBefore float64) {
	if consumeAmount <= 0 {
		return
	}
	bonus := balanceBefore - u.RechargedBalance
	if bonus < 0 {
		bonus = 0
	}
	fromBonus := consumeAmount
	if fromBonus > bonus {
		fromBonus = bonus
	}
	deduct := consumeAmount - fromBonus
	u.RechargedBalance -= deduct
	if u.RechargedBalance < 0 {
		u.RechargedBalance = 0
	}
}

func (s *MemoryStore) AccountTopup(ctx context.Context, userID string, amount float64, txType, note, stripePaymentID string, setHasPaid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersById[userID]
	if !ok {
		return ErrUserNotFound
	}
	before := u.Balance
	u.Balance += amount
	if isPaidTopupType(txType) {
		u.RechargedBalance += amount
	}
	if setHasPaid && !u.HasPaid {
		u.HasPaid = true
		now := time.Now()
		u.FirstPaidAt = &now
	}
	u.UpdatedAt = time.Now()
	s.recordAccountTxLocked(u, txType, amount, before, u.Balance, "", stripePaymentID)
	return nil
}

func (s *MemoryStore) AccountPreDeduct(ctx context.Context, userID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersById[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.Balance < amount {
		return ErrInsufficientBalance
	}
	before := u.Balance
	u.Balance -= amount
	s.memoryAdjustRechargedLocked(u, amount, before)
	return nil
}

func (s *MemoryStore) AccountRefundPreDeduct(ctx context.Context, userID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersById[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.Balance += amount
	return nil
}

func (s *MemoryStore) AccountFinalizeDeduct(ctx context.Context, userID, keyID, txType, modelName, requestID string, preDeducted, actualAmount float64, usage model.Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersById[userID]
	if !ok {
		return ErrUserNotFound
	}
	diff := actualAmount - preDeducted
	before := u.Balance + diff
	u.Balance -= diff
	if extra := actualAmount - preDeducted; extra > 0 {
		s.memoryAdjustRechargedLocked(u, extra, u.Balance+extra)
	}
	if keyID != "" {
		for _, k := range s.keys {
			if k.ID == keyID {
				k.SpentTotal += actualAmount
				break
			}
		}
	}
	s.recordAccountTxLocked(u, txType, actualAmount, before, u.Balance, keyID, "")
	return nil
}

func (s *MemoryStore) TryMonthlyGrant(ctx context.Context, userID string, grantUSD float64) (bool, error) {
	if grantUSD <= 0 {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersById[userID]
	if !ok {
		return false, ErrUserNotFound
	}
	month := time.Now().UTC().Format("2006-01")
	if u.LastMonthlyGrantMonth == month {
		return false, nil
	}
	before := u.Balance
	u.Balance += grantUSD
	u.LastMonthlyGrantMonth = month
	u.UpdatedAt = time.Now()
	s.recordAccountTxLocked(u, "monthly_grant", grantUSD, before, u.Balance, "", "")
	return true, nil
}

func (s *MemoryStore) GetKeySpentTotal(ctx context.Context, keyID string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.keys {
		if k.ID == keyID {
			return k.SpentTotal, nil
		}
	}
	return 0, nil
}

func (s *MemoryStore) CheckKeySpendLimit(ctx context.Context, keyID string, spendLimit *float64, additionalAmount float64) error {
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

func (s *MemoryStore) AddKeySpent(ctx context.Context, keyID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.keys {
		if k.ID == keyID {
			k.SpentTotal += amount
			return nil
		}
	}
	return nil
}

func (s *MemoryStore) SetKeySpendLimit(ctx context.Context, keyHash string, spendLimit *float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.keys[keyHash]
	if !ok {
		return ErrKeyNotFound
	}
	k.SpendLimit = spendLimit
	k.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) SetKeyAllowedModels(ctx context.Context, keyHash string, allowedModels []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	key.AllowedModels = allowedModels
	key.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) ListAccountTransactions(ctx context.Context, userID string, limit, offset int) ([]*model.Transaction, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*model.Transaction
	for i := range s.transactions {
		if s.transactions[i].UserID == userID {
			cp := s.transactions[i]
			out = append(out, &cp)
		}
	}
	total := len(out)
	if offset >= total {
		return []*model.Transaction{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return out[offset:end], total, nil
}

func (s *MemoryStore) recordAccountTxLocked(u *model.User, txType string, amount, before, after float64, keyID, stripeID string) {
	s.transactions = append(s.transactions, model.Transaction{
		ID:              generateTxID(),
		UserID:          u.ID,
		KeyID:           keyID,
		Type:            txType,
		Amount:          amount,
		BalanceBefore:   before,
		BalanceAfter:    after,
		StripePaymentID: stripeID,
		CreatedAt:       time.Now(),
	})
}
