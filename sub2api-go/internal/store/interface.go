package store

import (
	"context"
	"errors"

	"sub2api-go/internal/model"
)

// Errors
var (
	ErrKeyNotFound  = errors.New("api key not found")
	ErrKeyDisabled  = errors.New("api key is disabled")
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
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
}

// Ensure implementations satisfy the interface
var _ Store = (*MemoryStore)(nil)
var _ Store = (*RedisStore)(nil)
