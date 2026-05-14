package store

import (
	"context"
	"errors"
	"time"

	"sub2api-go/internal/model"
)

// Errors
var (
	ErrKeyNotFound  = errors.New("api key not found")
	ErrKeyDisabled  = errors.New("api key is disabled")
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")

	ErrRegisterOTPInvalid   = errors.New("invalid or expired verification code")
	ErrRegisterOTPCooldown   = errors.New("please wait before requesting another code")

	ErrResetPasswordOTPInvalid = errors.New("invalid or expired verification code")
	ErrResetPasswordOTPCooldown = errors.New("please wait before requesting another code")
)

// Store defines the interface for storage operations
type Store interface {
	// Key operations
	CreateKey(ctx context.Context, userID, name string, balance float64, rateLimit int) (string, *model.APIKey, error)
	ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error)
	GetKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	GetKeyByID(ctx context.Context, keyID string) (*model.APIKey, error)
	ListKeys(ctx context.Context, userID string) ([]*model.APIKey, error)
	UpdateKeySettings(ctx context.Context, keyHash string, ipWhitelist []string, rateLimit int) error
	DeleteKey(ctx context.Context, keyHash string) error

	// Balance operations
	GetBalance(ctx context.Context, keyHash string) (float64, error)
	PreDeduct(ctx context.Context, keyHash string, amount float64) error
	FinalizeDeduct(ctx context.Context, keyHash string, preDeducted, actualAmount float64, usage model.Usage, modelName, requestID string) error
	RefundPreDeduct(ctx context.Context, keyHash string, amount float64) error
	Topup(ctx context.Context, keyHash string, amount float64, note string) error

	// Usage
	GetUsageStats(ctx context.Context, keyHash string) (*model.UsageResponse, error)
	
	// Transactions
	ListTransactions(ctx context.Context, keyHash string, limit, offset int) ([]*model.Transaction, int, error)

	// User operations
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, userID string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error

	// Registration email OTP (6-digit code), consumed on successful register
	SaveRegisterOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error
	ConsumeRegisterOTP(ctx context.Context, email, plainCode string) error

	// Password reset email OTP (6-digit), consumed on successful reset
	SaveResetPasswordOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error
	ConsumeResetPasswordOTP(ctx context.Context, email, plainCode string) error

	// Analytics & audit (Dashboard + chat)
	AggregateConsumeByDay(ctx context.Context, keyHash string, days int) ([]model.DailyUsagePoint, error)
	AppendRequestLog(ctx context.Context, entry *model.RequestLogEntry) error
	ListRequestLogs(ctx context.Context, keyID string, limit, offset int) ([]*model.RequestLogEntry, int, error)
}

// Ensure implementations satisfy the interface
var _ Store = (*MemoryStore)(nil)
var _ Store = (*RedisStore)(nil)
