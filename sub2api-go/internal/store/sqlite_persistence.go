package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"sub2api-go/internal/model"
)

// SaveLedgerEntry writes one account ledger row (idempotent on id).
func (s *SQLiteStore) SaveLedgerEntry(ctx context.Context, tx *model.Transaction) error {
	if tx == nil || tx.ID == "" {
		return nil
	}
	actor := tx.Actor
	if actor == "" {
		actor = "system"
	}
	keyID := sql.NullString{String: tx.KeyID, Valid: tx.KeyID != ""}
	query := `
		INSERT INTO account_ledger (
			id, user_id, key_id, type, amount, balance_before, balance_after,
			model, input_tokens, output_tokens, request_id, stripe_payment_id, note, actor, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query,
		tx.ID,
		tx.UserID,
		keyID,
		tx.Type,
		tx.Amount,
		tx.BalanceBefore,
		tx.BalanceAfter,
		nullStr(tx.Model),
		nullInt(tx.InputTokens),
		nullInt(tx.OutputTokens),
		nullStr(tx.RequestID),
		nullStr(tx.StripePaymentID),
		nullStr(tx.Note),
		actor,
		tx.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) SaveRequestLog(ctx context.Context, entry *model.RequestLogEntry) error {
	if entry == nil || entry.ID == "" {
		return nil
	}
	stream := 0
	if entry.Stream {
		stream = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO request_logs (
			id, user_id, key_id, request_id, model, stream, outcome,
			input_tokens, output_tokens, charged_usd, ledger_entry_id, latency_ms, client_ip, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`,
		entry.ID,
		entry.UserID,
		entry.KeyID,
		nullStr(entry.RequestID),
		entry.Model,
		stream,
		entry.Outcome,
		nullInt(entry.InputTokens),
		nullInt(entry.OutputTokens),
		nullFloat(entry.ChargedUSD),
		nullStr(entry.LedgerEntryID),
		nullInt64(entry.LatencyMs),
		nullStr(entry.ClientIP),
		entry.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) SaveAdminAudit(ctx context.Context, id, adminID, targetUserID, action, beforeJSON, afterJSON, ledgerEntryID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_audit_log (id, admin_id, target_user_id, action, before_json, after_json, ledger_entry_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, nullStr(adminID), targetUserID, action, beforeJSON, afterJSON, nullStr(ledgerEntryID), time.Now().UTC(),
	)
	return err
}

func (s *SQLiteStore) EnqueueOutbox(ctx context.Context, id, entityType, entityID string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sync_outbox (id, entity_type, entity_id, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		id, entityType, entityID, string(b), time.Now().UTC(),
	)
	return err
}

type outboxRow struct {
	ID          string
	EntityType  string
	EntityID    string
	PayloadJSON string
	RetryCount  int
}

func (s *SQLiteStore) ListPendingOutbox(ctx context.Context, limit int) ([]outboxRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entity_type, entity_id, payload_json, retry_count
		FROM sync_outbox WHERE processed_at IS NULL
		ORDER BY created_at ASC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.ID, &r.EntityType, &r.EntityID, &r.PayloadJSON, &r.RetryCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) MarkOutboxProcessed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sync_outbox SET processed_at = ? WHERE id = ?`, time.Now().UTC(), id)
	return err
}

func (s *SQLiteStore) MarkOutboxFailed(ctx context.Context, id, lastErr string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sync_outbox SET retry_count = retry_count + 1, last_error = ? WHERE id = ?
	`, lastErr, id)
	return err
}

// GetUserAccountSnapshot reads wallet fields from SQLite (for admin reload-from-db).
func (s *SQLiteStore) GetUserAccountSnapshot(ctx context.Context, userID string) (spendable, recharged float64, hasPaid bool, firstPaidAt *time.Time, lastGrantMonth string, status string, err error) {
	var fp sql.NullTime
	var hasPaidInt int
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(spendable_balance, balance, 0), COALESCE(recharged_balance, 0),
		       has_paid, first_paid_at, last_monthly_grant_month, status
		FROM users WHERE id = ?
	`, userID).Scan(&spendable, &recharged, &hasPaidInt, &fp, &lastGrantMonth, &status)
	if err == sql.ErrNoRows {
		return 0, 0, false, nil, "", "", ErrUserNotFound
	}
	if err != nil {
		return
	}
	hasPaid = hasPaidInt != 0
	if fp.Valid {
		t := fp.Time
		firstPaidAt = &t
	}
	return
}

func (s *SQLiteStore) ListUsers(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, password_hash, name, status, email_verified,
		       COALESCE(spendable_balance, balance, 0), COALESCE(recharged_balance, 0),
		       has_paid, first_paid_at, last_monthly_grant_month, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		var u model.User
		var hasPaid int
		var fp sql.NullTime
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Status, &u.EmailVerified,
			&u.Balance, &u.RechargedBalance, &hasPaid, &fp, &u.LastMonthlyGrantMonth,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		u.HasPaid = hasPaid != 0
		if fp.Valid {
			u.FirstPaidAt = &fp.Time
		}
		users = append(users, &u)
	}
	return users, total, rows.Err()
}

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt(n int) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(n), Valid: true}
}

func nullInt64(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

func nullFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}
