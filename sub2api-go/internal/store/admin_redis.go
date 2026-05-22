package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"sub2api-go/internal/model"
)

var ErrAdminNotSupported = errors.New("admin operations require Redis store with SQLite for full admin features")

func (s *MemoryStore) AdminListUsers(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	return nil, 0, ErrAdminNotSupported
}

func (s *MemoryStore) AdminGetUser(ctx context.Context, userID string) (*model.User, error) {
	return nil, ErrAdminNotSupported
}

func (s *MemoryStore) AdminAdjustUserBalance(ctx context.Context, userID string, req model.AdminAdjustBalanceRequest) (*model.Transaction, error) {
	return nil, ErrAdminNotSupported
}

func (s *MemoryStore) AdminSetUserStatus(ctx context.Context, userID string, status, note string) error {
	return ErrAdminNotSupported
}

func (s *MemoryStore) AdminReloadUserFromDB(ctx context.Context, userID string) error {
	return ErrAdminNotSupported
}

func (s *RedisStore) AdminListUsers(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	if s.sqlite != nil {
		return s.sqlite.ListUsers(ctx, limit, offset)
	}
	return s.adminListUsersFromRedis(ctx, limit, offset)
}

func (s *RedisStore) adminListUsersFromRedis(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	var cursor uint64
	var all []*model.User
	for {
		keys, next, err := s.client.Scan(ctx, cursor, KeyPrefixUser+"*", 100).Result()
		if err != nil {
			return nil, 0, err
		}
		for _, k := range keys {
			raw, err := s.client.Get(ctx, k).Result()
			if err != nil {
				continue
			}
			var u model.User
			if json.Unmarshal([]byte(raw), &u) != nil {
				continue
			}
			if bal, err := s.GetAccountBalance(ctx, u.ID); err == nil {
				u.Balance = bal
			}
			if rec, err := s.GetAccountRechargedBalance(ctx, u.ID); err == nil {
				u.RechargedBalance = rec
			}
			all = append(all, &u)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	total := len(all)
	if offset >= total {
		return []*model.User{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (s *RedisStore) AdminGetUser(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.RechargedBalance, _ = s.GetAccountRechargedBalance(ctx, userID)
	return u, nil
}

func (s *RedisStore) AdminAdjustUserBalance(ctx context.Context, userID string, req model.AdminAdjustBalanceRequest) (*model.Transaction, error) {
	oldSpendable, err := s.GetAccountBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	oldRecharged, _ := s.GetAccountRechargedBalance(ctx, userID)

	var newSpendable float64
	switch {
	case req.SpendableBalance != nil:
		newSpendable = *req.SpendableBalance
	case req.AdjustAmount != nil:
		newSpendable = oldSpendable + *req.AdjustAmount
	default:
		return nil, fmt.Errorf("provide spendable_balance or adjust_amount")
	}
	if newSpendable < 0 {
		newSpendable = 0
	}

	delta := newSpendable - oldSpendable
	if err := s.client.Set(ctx, accountBalanceKey(userID), fmt.Sprintf("%.6f", newSpendable), 0).Err(); err != nil {
		return nil, err
	}

	newRecharged := oldRecharged
	if req.RechargedBalance != nil {
		newRecharged = *req.RechargedBalance
		if newRecharged < 0 {
			newRecharged = 0
		}
		_ = s.client.Set(ctx, accountRechargedKey(userID), fmt.Sprintf("%.6f", newRecharged), 0).Err()
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Balance = newSpendable
	user.RechargedBalance = newRecharged
	user.UpdatedAt = time.Now().UTC()
	_ = s.UpdateUser(ctx, user)
	s.writeThroughUserAccount(ctx, user)

	now := time.Now().UTC()
	tx := &model.Transaction{
		ID:            generateTxID(),
		UserID:        userID,
		Type:          "admin_adjust",
		Amount:        delta,
		BalanceBefore: oldSpendable,
		BalanceAfter:  newSpendable,
		Note:          req.Note,
		Actor:         "admin",
		CreatedAt:     now,
	}
	txJSON, _ := json.Marshal(tx)
	_ = s.client.Set(ctx, KeyPrefixTransaction+tx.ID, txJSON, TransactionRedisTTL).Err()
	s.writeThroughLedger(ctx, tx)

	if s.sqlite != nil {
		before, _ := json.Marshal(map[string]float64{"spendable": oldSpendable, "recharged": oldRecharged})
		after, _ := json.Marshal(map[string]float64{"spendable": newSpendable, "recharged": newRecharged})
		_ = s.sqlite.SaveAdminAudit(ctx, generateTxID(), "", userID, "adjust_balance", string(before), string(after), tx.ID)
	}
	return tx, nil
}

func (s *RedisStore) AdminSetUserStatus(ctx context.Context, userID, status, note string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	beforeStatus := user.Status
	user.Status = status
	user.UpdatedAt = time.Now().UTC()
	if err := s.UpdateUser(ctx, user); err != nil {
		return err
	}
	s.writeThroughUserAccount(ctx, user)

	if s.sqlite != nil {
		before, _ := json.Marshal(map[string]string{"status": beforeStatus})
		after, _ := json.Marshal(map[string]string{"status": status, "note": note})
		_ = s.sqlite.SaveAdminAudit(ctx, generateTxID(), "", userID, "set_status", string(before), string(after), "")
	}
	return nil
}

func (s *RedisStore) AdminReloadUserFromDB(ctx context.Context, userID string) error {
	if s.sqlite == nil {
		return fmt.Errorf("sqlite not configured")
	}
	spendable, recharged, hasPaid, firstPaidAt, lastGrant, status, err := s.sqlite.GetUserAccountSnapshot(ctx, userID)
	if err != nil {
		return err
	}
	_ = s.client.Set(ctx, accountBalanceKey(userID), fmt.Sprintf("%.6f", spendable), 0).Err()
	_ = s.client.Set(ctx, accountRechargedKey(userID), fmt.Sprintf("%.6f", recharged), 0).Err()

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Balance = spendable
	user.RechargedBalance = recharged
	user.HasPaid = hasPaid
	user.FirstPaidAt = firstPaidAt
	user.LastMonthlyGrantMonth = lastGrant
	if status != "" {
		user.Status = status
	}
	user.UpdatedAt = time.Now().UTC()
	userJSON, _ := json.Marshal(user)
	_ = s.client.Set(ctx, KeyPrefixUser+userID, userJSON, 0).Err()
	return nil
}
