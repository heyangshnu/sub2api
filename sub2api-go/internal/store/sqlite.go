package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"sub2api-go/internal/model"
)

// SQLiteStore implements persistent storage using SQLite
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		email_verified INTEGER NOT NULL DEFAULT 0,
		email_verify_token_hash TEXT,
		email_verify_expires_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_verify_token_hash ON users(email_verify_token_hash);

	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT UNIQUE NOT NULL,
		key_prefix TEXT NOT NULL,
		user_id TEXT NOT NULL,
		name TEXT,
		balance REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		rate_limit INTEGER DEFAULT 60,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);

	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		key_id TEXT NOT NULL,
		type TEXT NOT NULL,
		amount REAL NOT NULL,
		balance_before REAL NOT NULL,
		balance_after REAL NOT NULL,
		model TEXT,
		input_tokens INTEGER,
		output_tokens INTEGER,
		request_id TEXT,
		stripe_payment_id TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (key_id) REFERENCES api_keys(id)
	);
	CREATE INDEX IF NOT EXISTS idx_transactions_key_id ON transactions(key_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);

	CREATE TABLE IF NOT EXISTS model_pricing (
		model TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		input_price_per_k REAL NOT NULL,
		output_price_per_k REAL NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS register_otps (
		email TEXT PRIMARY KEY,
		code_hash TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS reset_password_otps (
		email TEXT PRIMARY KEY,
		code_hash TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Backward-compatible migrations for existing databases.
	alterStmts := []string{
		`ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN email_verify_token_hash TEXT`,
		`ALTER TABLE users ADD COLUMN email_verify_expires_at DATETIME`,
		`ALTER TABLE users ADD COLUMN balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN has_paid INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN first_paid_at DATETIME`,
		`ALTER TABLE users ADD COLUMN last_monthly_grant_month TEXT`,
		`ALTER TABLE api_keys ADD COLUMN spend_limit REAL`,
		`ALTER TABLE api_keys ADD COLUMN spent_total REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE transactions ADD COLUMN user_id TEXT`,
	}
	for _, stmt := range alterStmts {
		if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_verify_token_hash ON users(email_verify_token_hash)`)
	return nil
}

// ==================== Key Operations ====================

func (s *SQLiteStore) SaveKey(ctx context.Context, key *model.APIKey) error {
	query := `
		INSERT INTO api_keys (id, key_hash, key_prefix, user_id, name, balance, status, rate_limit, created_at, updated_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			balance = excluded.balance,
			status = excluded.status,
			updated_at = excluded.updated_at,
			last_used_at = excluded.last_used_at
	`

	_, err := s.db.ExecContext(ctx, query,
		key.ID,
		key.KeyHash,
		key.KeyPrefix,
		key.UserID,
		key.Name,
		key.Balance,
		key.Status,
		key.RateLimit,
		key.CreatedAt,
		key.UpdatedAt,
		key.LastUsedAt,
	)
	return err
}

func (s *SQLiteStore) GetKeyByID(ctx context.Context, keyID string) (*model.APIKey, error) {
	query := `
		SELECT id, key_hash, key_prefix, user_id, name, balance, status, rate_limit, created_at, updated_at, last_used_at
		FROM api_keys WHERE id = ?
	`

	var key model.APIKey
	var lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, keyID).Scan(
		&key.ID,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.UserID,
		&key.Name,
		&key.Balance,
		&key.Status,
		&key.RateLimit,
		&key.CreatedAt,
		&key.UpdatedAt,
		&lastUsedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

func (s *SQLiteStore) GetKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `
		SELECT id, key_hash, key_prefix, user_id, name, balance, status, rate_limit, created_at, updated_at, last_used_at
		FROM api_keys WHERE key_hash = ?
	`

	var key model.APIKey
	var lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, keyHash).Scan(
		&key.ID,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.UserID,
		&key.Name,
		&key.Balance,
		&key.Status,
		&key.RateLimit,
		&key.CreatedAt,
		&key.UpdatedAt,
		&lastUsedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

func (s *SQLiteStore) ListKeys(ctx context.Context, userID string) ([]*model.APIKey, error) {
	var query string
	var args []interface{}

	if userID != "" {
		query = `SELECT id, key_hash, key_prefix, user_id, name, balance, status, rate_limit, created_at, updated_at, last_used_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`
		args = []interface{}{userID}
	} else {
		query = `SELECT id, key_hash, key_prefix, user_id, name, balance, status, rate_limit, created_at, updated_at, last_used_at FROM api_keys ORDER BY created_at DESC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		var key model.APIKey
		var lastUsedAt sql.NullTime

		if err := rows.Scan(
			&key.ID,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.UserID,
			&key.Name,
			&key.Balance,
			&key.Status,
			&key.RateLimit,
			&key.CreatedAt,
			&key.UpdatedAt,
			&lastUsedAt,
		); err != nil {
			return nil, err
		}

		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		keys = append(keys, &key)
	}

	return keys, rows.Err()
}

func (s *SQLiteStore) UpdateBalance(ctx context.Context, keyID string, balance float64) error {
	query := `UPDATE api_keys SET balance = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, balance, time.Now(), keyID)
	return err
}

// ==================== Transaction Operations ====================

func (s *SQLiteStore) SaveTransaction(ctx context.Context, tx *model.Transaction) error {
	query := `
		INSERT INTO transactions (id, key_id, type, amount, balance_before, balance_after, model, input_tokens, output_tokens, request_id, stripe_payment_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		tx.ID,
		tx.KeyID,
		tx.Type,
		tx.Amount,
		tx.BalanceBefore,
		tx.BalanceAfter,
		tx.Model,
		tx.InputTokens,
		tx.OutputTokens,
		tx.RequestID,
		tx.StripePaymentID,
		tx.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetTransactionsByKeyID(ctx context.Context, keyID string, limit int) ([]*model.Transaction, error) {
	query := `
		SELECT id, key_id, type, amount, balance_before, balance_after, model, input_tokens, output_tokens, request_id, stripe_payment_id, created_at
		FROM transactions WHERE key_id = ? ORDER BY created_at DESC LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, keyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		var tx model.Transaction
		var stripePaymentID sql.NullString

		if err := rows.Scan(
			&tx.ID,
			&tx.KeyID,
			&tx.Type,
			&tx.Amount,
			&tx.BalanceBefore,
			&tx.BalanceAfter,
			&tx.Model,
			&tx.InputTokens,
			&tx.OutputTokens,
			&tx.RequestID,
			&stripePaymentID,
			&tx.CreatedAt,
		); err != nil {
			return nil, err
		}

		if stripePaymentID.Valid {
			tx.StripePaymentID = stripePaymentID.String
		}
		transactions = append(transactions, &tx)
	}

	return transactions, rows.Err()
}

func (s *SQLiteStore) GetUsageStatsByKeyID(ctx context.Context, keyID string) (totalUsed float64, requestCount int, lastRequestAt *time.Time, err error) {
	query := `
		SELECT COALESCE(SUM(amount), 0), COUNT(*), MAX(created_at)
		FROM transactions WHERE key_id = ? AND type = 'consume'
	`

	var lastReq sql.NullTime
	err = s.db.QueryRowContext(ctx, query, keyID).Scan(&totalUsed, &requestCount, &lastReq)
	if err != nil {
		return
	}

	if lastReq.Valid {
		lastRequestAt = &lastReq.Time
	}
	return
}

// ==================== User Operations ====================

func (s *SQLiteStore) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, status, email_verified, email_verify_token_hash, email_verify_expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Status,
		user.EmailVerified,
		user.EmailVerifyTokenHash,
		user.EmailVerifyExpiresAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, status, email_verified, email_verify_token_hash, email_verify_expires_at, created_at, updated_at
		FROM users WHERE email = ?
	`
	var user model.User
	var verifyExpiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Status,
		&user.EmailVerified,
		&user.EmailVerifyTokenHash,
		&verifyExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if verifyExpiresAt.Valid {
		user.EmailVerifyExpiresAt = &verifyExpiresAt.Time
	}
	return &user, nil
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, status, email_verified, email_verify_token_hash, email_verify_expires_at, created_at, updated_at
		FROM users WHERE id = ?
	`
	var user model.User
	var verifyExpiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Status,
		&user.EmailVerified,
		&user.EmailVerifyTokenHash,
		&verifyExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if verifyExpiresAt.Valid {
		user.EmailVerifyExpiresAt = &verifyExpiresAt.Time
	}
	return &user, nil
}

func (s *SQLiteStore) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET email = ?, password_hash = ?, name = ?, status = ?, email_verified = ?, email_verify_token_hash = ?, email_verify_expires_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Status,
		user.EmailVerified,
		user.EmailVerifyTokenHash,
		user.EmailVerifyExpiresAt,
		user.UpdatedAt,
		user.ID,
	)
	return err
}

// SaveUserAccount persists account wallet fields to SQLite (Redis remains source of truth at runtime).
func (s *SQLiteStore) SaveUserAccount(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET balance = ?, has_paid = ?, first_paid_at = ?, last_monthly_grant_month = ?, updated_at = ?
		WHERE id = ?
	`
	hasPaid := 0
	if user.HasPaid {
		hasPaid = 1
	}
	_, err := s.db.ExecContext(ctx, query,
		user.Balance,
		hasPaid,
		user.FirstPaidAt,
		user.LastMonthlyGrantMonth,
		user.UpdatedAt,
		user.ID,
	)
	return err
}

func normalizeRegisterEmailSQLite(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *SQLiteStore) SaveRegisterOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error {
	em := normalizeRegisterEmailSQLite(email)
	var prevCreated time.Time
	err := s.db.QueryRowContext(ctx, `SELECT created_at FROM register_otps WHERE email = ?`, em).Scan(&prevCreated)
	if err == nil {
		if time.Since(prevCreated) < 60*time.Second {
			return ErrRegisterOTPCooldown
		}
	} else if err != sql.ErrNoRows {
		return err
	}

	q := `
		INSERT INTO register_otps (email, code_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			code_hash = excluded.code_hash,
			expires_at = excluded.expires_at,
			created_at = excluded.created_at
	`
	_, err = s.db.ExecContext(ctx, q, em, codeHash, expiresAt, createdAt)
	return err
}

func (s *SQLiteStore) ConsumeRegisterOTP(ctx context.Context, email, plainCode string) error {
	em := normalizeRegisterEmailSQLite(email)
	var hash string
	var exp time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT code_hash, expires_at FROM register_otps WHERE email = ?`, em,
	).Scan(&hash, &exp)
	if err == sql.ErrNoRows {
		return ErrRegisterOTPInvalid
	}
	if err != nil {
		return err
	}
	if time.Now().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM register_otps WHERE email = ?`, em)
		return ErrRegisterOTPInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.TrimSpace(plainCode))); err != nil {
		return ErrRegisterOTPInvalid
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM register_otps WHERE email = ?`, em)
	return err
}

func (s *SQLiteStore) SaveResetPasswordOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error {
	em := normalizeRegisterEmailSQLite(email)
	var prevCreated time.Time
	err := s.db.QueryRowContext(ctx, `SELECT created_at FROM reset_password_otps WHERE email = ?`, em).Scan(&prevCreated)
	if err == nil {
		if time.Since(prevCreated) < 60*time.Second {
			return ErrResetPasswordOTPCooldown
		}
	} else if err != sql.ErrNoRows {
		return err
	}

	q := `
		INSERT INTO reset_password_otps (email, code_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			code_hash = excluded.code_hash,
			expires_at = excluded.expires_at,
			created_at = excluded.created_at
	`
	_, err = s.db.ExecContext(ctx, q, em, codeHash, expiresAt, createdAt)
	return err
}

func (s *SQLiteStore) ConsumeResetPasswordOTP(ctx context.Context, email, plainCode string) error {
	em := normalizeRegisterEmailSQLite(email)
	var hash string
	var exp time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT code_hash, expires_at FROM reset_password_otps WHERE email = ?`, em,
	).Scan(&hash, &exp)
	if err == sql.ErrNoRows {
		return ErrResetPasswordOTPInvalid
	}
	if err != nil {
		return err
	}
	if time.Now().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM reset_password_otps WHERE email = ?`, em)
		return ErrResetPasswordOTPInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.TrimSpace(plainCode))); err != nil {
		return ErrResetPasswordOTPInvalid
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM reset_password_otps WHERE email = ?`, em)
	return err
}

// DeleteRegistrationByEmail removes user, keys, transactions, and register / reset OTP rows in SQLite.
// linkedUserID is set when the account lived in Redis but api_keys were synced to SQLite without a users row.
func (s *SQLiteStore) DeleteRegistrationByEmail(ctx context.Context, email string, linkedUserID string) error {
	em := normalizeRegisterEmailSQLite(email)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var uid string
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email) = lower(?)`, email).Scan(&uid)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if uid == "" && linkedUserID != "" {
		uid = linkedUserID
	}
	if uid != "" {
		if _, err = tx.ExecContext(ctx, `DELETE FROM transactions WHERE key_id IN (SELECT id FROM api_keys WHERE user_id = ?)`, uid); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = ?`, uid); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, uid); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM register_otps WHERE email = ?`, em); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM reset_password_otps WHERE email = ?`, em); err != nil {
		return err
	}
	return tx.Commit()
}
