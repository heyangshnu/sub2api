package store

import (
	"context"
	"log"

	"sub2api-go/internal/model"
)

// writeThroughLedger persists a transaction to SQLite; on failure enqueues sync_outbox.
func (s *RedisStore) writeThroughLedger(ctx context.Context, tx *model.Transaction) {
	if s.sqlite == nil || tx == nil {
		return
	}
	if err := s.sqlite.SaveLedgerEntry(ctx, tx); err != nil {
		log.Printf("[persist] ledger %s: %v", tx.ID, err)
		_ = s.sqlite.EnqueueOutbox(ctx, "outbox-"+tx.ID, "ledger", tx.ID, tx)
	}
}

func (s *RedisStore) writeThroughRequestLog(ctx context.Context, entry *model.RequestLogEntry) {
	if s.sqlite == nil || entry == nil {
		return
	}
	if entry.UserID == "" && entry.KeyID != "" {
		if key, err := s.GetKeyByID(ctx, entry.KeyID); err == nil {
			entry.UserID = key.UserID
		}
	}
	if err := s.sqlite.SaveRequestLog(ctx, entry); err != nil {
		log.Printf("[persist] request_log %s: %v", entry.ID, err)
		_ = s.sqlite.EnqueueOutbox(ctx, "outbox-"+entry.ID, "request_log", entry.ID, entry)
	}
}

func (s *RedisStore) writeThroughUserAccount(ctx context.Context, user *model.User) {
	if s.sqlite == nil || user == nil {
		return
	}
	spendable, _ := s.GetAccountBalance(ctx, user.ID)
	recharged, _ := s.GetAccountRechargedBalance(ctx, user.ID)
	if err := s.sqlite.SaveUserAccount(ctx, user, spendable, recharged); err != nil {
		log.Printf("[persist] user account %s: %v", user.ID, err)
		_ = s.sqlite.EnqueueOutbox(ctx, "outbox-user-"+user.ID, "user_account", user.ID, map[string]interface{}{
			"user_id": user.ID, "spendable": spendable, "recharged": recharged,
		})
	}
}

func (s *RedisStore) writeThroughKey(ctx context.Context, key *model.APIKey) {
	if s.sqlite == nil || key == nil {
		return
	}
	if err := s.sqlite.SaveKey(ctx, key); err != nil {
		log.Printf("[persist] api_key %s: %v", key.ID, err)
		_ = s.sqlite.EnqueueOutbox(ctx, "outbox-key-"+key.ID, "api_key", key.ID, key)
	}
}
