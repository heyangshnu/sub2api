package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"sub2api-go/internal/model"
)

// Redis key prefixes
const (
	KeyPrefixAPIKey      = "apikey:"       // apikey:<hash> -> JSON
	KeyPrefixBalance     = "balance:"      // balance:<hash> -> float string
	KeyPrefixTransaction = "tx:"           // tx:<id> -> JSON
	KeyPrefixUserKeys    = "user_keys:"    // user_keys:<user_id> -> SET of key hashes
	KeyPrefixKeyCounter  = "key_counter"   // atomic counter for key generation
	KeyPrefixUser        = "user:"         // user:<id> -> JSON
	KeyPrefixUserEmail   = "user_email:"   // user_email:<email> -> user_id
)

// Lua script for atomic pre-deduct
// Returns: 1 = success, 0 = insufficient balance, -1 = key not found
const luaPreDeduct = `
local balance_key = KEYS[1]
local amount = tonumber(ARGV[1])

local balance = redis.call('GET', balance_key)
if not balance then
    return -1
end

balance = tonumber(balance)
if balance < amount then
    return 0
end

redis.call('SET', balance_key, tostring(balance - amount))
return 1
`

// Lua script for atomic finalize deduct (adjust from pre-deducted to actual)
const luaFinalizeDeduct = `
local balance_key = KEYS[1]
local pre_deducted = tonumber(ARGV[1])
local actual = tonumber(ARGV[2])

local balance = redis.call('GET', balance_key)
if not balance then
    return -1
end

balance = tonumber(balance)
local diff = actual - pre_deducted
balance = balance - diff
redis.call('SET', balance_key, tostring(balance))
return balance
`

// Lua script for atomic refund
const luaRefund = `
local balance_key = KEYS[1]
local amount = tonumber(ARGV[1])

local balance = redis.call('GET', balance_key)
if not balance then
    return -1
end

balance = tonumber(balance) + amount
redis.call('SET', balance_key, tostring(balance))
return balance
`

// RedisStore implements storage using Redis
type RedisStore struct {
	client  *redis.Client
	scripts map[string]*redis.Script
	sqlite  *SQLiteStore // For user data persistence
}

func NewRedisStore(redisURL string, sqlite *SQLiteStore) (*RedisStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	store := &RedisStore{
		client: client,
		scripts: map[string]*redis.Script{
			"pre_deduct":      redis.NewScript(luaPreDeduct),
			"finalize_deduct": redis.NewScript(luaFinalizeDeduct),
			"refund":          redis.NewScript(luaRefund),
		},
		sqlite: sqlite,
	}

	return store, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Client() *redis.Client {
	return s.client
}

// ==================== API Key Operations ====================

func (s *RedisStore) GenerateAPIKey(ctx context.Context) (string, error) {
	counter, err := s.client.Incr(ctx, KeyPrefixKeyCounter).Result()
	if err != nil {
		return "", err
	}
	
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%d:%d", counter, timestamp)
	h := sha256.Sum256([]byte(data))
	random := hex.EncodeToString(h[:16])
	
	return "sk-sub2api-" + random, nil
}

func (s *RedisStore) CreateKey(ctx context.Context, userID, name string, balance float64, rateLimit int) (string, *model.APIKey, error) {
	rawKey, err := s.GenerateAPIKey(ctx)
	if err != nil {
		return "", nil, err
	}
	
	keyHash := HashKey(rawKey)
	now := time.Now()
	
	key := &model.APIKey{
		ID:        keyHash[:16],
		KeyHash:   keyHash,
		KeyPrefix: rawKey[:20] + "...",
		UserID:    userID,
		Name:      name,
		Balance:   balance,
		Status:    "active",
		RateLimit: rateLimit,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	// Store key metadata
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return "", nil, err
	}
	
	pipe := s.client.Pipeline()
	pipe.Set(ctx, KeyPrefixAPIKey+keyHash, keyJSON, 0)
	pipe.Set(ctx, KeyPrefixBalance+keyHash, fmt.Sprintf("%.6f", balance), 0)
	pipe.SAdd(ctx, KeyPrefixUserKeys+userID, keyHash)
	
	if _, err := pipe.Exec(ctx); err != nil {
		return "", nil, err
	}
	
	return rawKey, key, nil
}

func (s *RedisStore) ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := HashKey(rawKey)
	
	keyJSON, err := s.client.Get(ctx, KeyPrefixAPIKey+keyHash).Result()
	if err == redis.Nil {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	
	var key model.APIKey
	if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
		return nil, err
	}
	
	// KeyHash is not serialized (json:"-"), set it manually
	key.KeyHash = keyHash
	
	if key.Status != "active" {
		return nil, ErrKeyDisabled
	}
	
	return &key, nil
}

func (s *RedisStore) GetKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	keyJSON, err := s.client.Get(ctx, KeyPrefixAPIKey+keyHash).Result()
	if err == redis.Nil {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	
	var key model.APIKey
	if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
		return nil, err
	}
	
	// KeyHash is not serialized (json:"-"), set it manually
	key.KeyHash = keyHash
	
	// Get current balance from balance key (source of truth)
	balanceStr, err := s.client.Get(ctx, KeyPrefixBalance+keyHash).Result()
	if err == nil {
		if balance, err := strconv.ParseFloat(balanceStr, 64); err == nil {
			key.Balance = balance
		}
	}
	
	return &key, nil
}

func (s *RedisStore) GetKeyByID(ctx context.Context, keyID string) (*model.APIKey, error) {
	// This requires scanning - in production, maintain an index
	// For now, iterate through all keys
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, KeyPrefixAPIKey+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		
		for _, k := range keys {
			keyJSON, err := s.client.Get(ctx, k).Result()
			if err != nil {
				continue
			}
			
			var key model.APIKey
			if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
				continue
			}
			
			if key.ID == keyID {
				// Get current balance
				balanceStr, err := s.client.Get(ctx, KeyPrefixBalance+key.KeyHash).Result()
				if err == nil {
					if balance, err := strconv.ParseFloat(balanceStr, 64); err == nil {
						key.Balance = balance
					}
				}
				return &key, nil
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	return nil, ErrKeyNotFound
}

func (s *RedisStore) ListKeys(ctx context.Context, userID string) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	
	if userID != "" {
		// Get keys for specific user
		keyHashes, err := s.client.SMembers(ctx, KeyPrefixUserKeys+userID).Result()
		if err != nil {
			return nil, err
		}
		
		for _, hash := range keyHashes {
			key, err := s.GetKeyByHash(ctx, hash)
			if err == nil {
				keys = append(keys, key)
			}
		}
	} else {
		// Get all keys (scan)
		var cursor uint64
		for {
			keyNames, nextCursor, err := s.client.Scan(ctx, cursor, KeyPrefixAPIKey+"*", 100).Result()
			if err != nil {
				return nil, err
			}
			
			for _, k := range keyNames {
				keyJSON, err := s.client.Get(ctx, k).Result()
				if err != nil {
					continue
				}
				
				var key model.APIKey
				if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
					continue
				}
				
				// Get current balance
				balanceStr, err := s.client.Get(ctx, KeyPrefixBalance+key.KeyHash).Result()
				if err == nil {
					if balance, err := strconv.ParseFloat(balanceStr, 64); err == nil {
						key.Balance = balance
					}
				}
				
				keys = append(keys, &key)
			}
			
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}
	
	return keys, nil
}

// UpdateKeySettings 更新 Key 的 IP 白名单和频次限制
func (s *RedisStore) UpdateKeySettings(ctx context.Context, keyHash string, ipWhitelist []string, rateLimit int) error {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return err
	}
	
	if ipWhitelist != nil {
		key.IPWhitelist = ipWhitelist
	}
	if rateLimit > 0 {
		key.RateLimit = rateLimit
	}
	key.UpdatedAt = time.Now()
	
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return err
	}
	
	return s.client.Set(ctx, KeyPrefixAPIKey+keyHash, keyJSON, 0).Err()
}

// DeleteKey 删除 Key
func (s *RedisStore) DeleteKey(ctx context.Context, keyHash string) error {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return err
	}
	
	pipe := s.client.Pipeline()
	pipe.Del(ctx, KeyPrefixAPIKey+keyHash)
	pipe.Del(ctx, KeyPrefixBalance+keyHash)
	pipe.SRem(ctx, KeyPrefixUserKeys+key.UserID, keyHash)
	
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStore) GetBalance(ctx context.Context, keyHash string) (float64, error) {
	balanceStr, err := s.client.Get(ctx, KeyPrefixBalance+keyHash).Result()
	if err == redis.Nil {
		return 0, ErrKeyNotFound
	}
	if err != nil {
		return 0, err
	}
	
	return strconv.ParseFloat(balanceStr, 64)
}

func (s *RedisStore) PreDeduct(ctx context.Context, keyHash string, amount float64) error {
	balanceKey := KeyPrefixBalance + keyHash
	
	result, err := s.scripts["pre_deduct"].Run(ctx, s.client,
		[]string{balanceKey},
		fmt.Sprintf("%.6f", amount),
	).Int()
	
	if err != nil {
		return err
	}
	
	switch result {
	case 1:
		return nil
	case 0:
		return ErrInsufficientBalance
	case -1:
		return ErrKeyNotFound
	default:
		return errors.New("unexpected result from pre_deduct script")
	}
}

func (s *RedisStore) FinalizeDeduct(ctx context.Context, keyHash string, preDeducted, actualAmount float64, usage model.Usage, modelName, requestID string) error {
	_, err := s.scripts["finalize_deduct"].Run(ctx, s.client,
		[]string{KeyPrefixBalance + keyHash},
		fmt.Sprintf("%.6f", preDeducted),
		fmt.Sprintf("%.6f", actualAmount),
	).Float64()
	
	if err != nil {
		return err
	}
	
	// Get key info for transaction record
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return nil // Balance updated, transaction record is secondary
	}
	
	// Update key metadata (last used time)
	now := time.Now()
	key.LastUsedAt = &now
	key.UpdatedAt = now
	keyJSON, _ := json.Marshal(key)
	s.client.Set(ctx, KeyPrefixAPIKey+keyHash, keyJSON, 0)
	
	// Record transaction (fire and forget for now, will be persisted to DB later)
	tx := model.Transaction{
		ID:            generateTxID(),
		KeyID:         key.ID,
		Type:          "consume",
		Amount:        actualAmount,
		BalanceBefore: key.Balance + actualAmount, // Approximate
		BalanceAfter:  key.Balance,
		Model:         modelName,
		InputTokens:   usage.PromptTokens,
		OutputTokens:  usage.CompletionTokens,
		RequestID:     requestID,
		CreatedAt:     now,
	}
	txJSON, _ := json.Marshal(tx)
	s.client.Set(ctx, KeyPrefixTransaction+tx.ID, txJSON, 24*time.Hour) // TTL 24h, will sync to DB
	
	return nil
}

func (s *RedisStore) RefundPreDeduct(ctx context.Context, keyHash string, amount float64) error {
	_, err := s.scripts["refund"].Run(ctx, s.client,
		[]string{KeyPrefixBalance + keyHash},
		fmt.Sprintf("%.6f", amount),
	).Float64()
	
	return err
}

func (s *RedisStore) Topup(ctx context.Context, keyHash string, amount float64, note string) error {
	// Get current balance first for transaction record
	oldBalance, err := s.GetBalance(ctx, keyHash)
	if err != nil {
		return err
	}
	
	// Atomic add
	newBalance, err := s.client.IncrByFloat(ctx, KeyPrefixBalance+keyHash, amount).Result()
	if err != nil {
		return err
	}
	
	// Record transaction
	now := time.Now()
	key, _ := s.GetKeyByHash(ctx, keyHash)
	
	tx := model.Transaction{
		ID:            generateTxID(),
		KeyID:         key.ID,
		Type:          "topup",
		Amount:        amount,
		BalanceBefore: oldBalance,
		BalanceAfter:  newBalance,
		CreatedAt:     now,
	}
	txJSON, _ := json.Marshal(tx)
	s.client.Set(ctx, KeyPrefixTransaction+tx.ID, txJSON, 24*time.Hour)
	
	return nil
}

// ==================== Usage Stats ====================

func (s *RedisStore) GetUsageStats(ctx context.Context, keyHash string) (*model.UsageResponse, error) {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	
	// Get balance
	balance, _ := s.GetBalance(ctx, keyHash)
	
	// Count transactions (simplified - scan tx keys)
	var totalUsed float64
	var requestCount int
	var lastRequest time.Time
	
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 100).Result()
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
			
			if tx.KeyID == key.ID && tx.Type == "consume" {
				totalUsed += tx.Amount
				requestCount++
				if tx.CreatedAt.After(lastRequest) {
					lastRequest = tx.CreatedAt
				}
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	resp := &model.UsageResponse{
		Balance:      balance,
		TotalUsed:    totalUsed,
		RequestCount: requestCount,
	}
	
	if !lastRequest.IsZero() {
		resp.LastRequestAt = lastRequest.Format(time.RFC3339)
	}
	
	return resp, nil
}

// ListTransactions returns transactions for a key with pagination
func (s *RedisStore) ListTransactions(ctx context.Context, keyHash string, limit, offset int) ([]*model.Transaction, int, error) {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, 0, err
	}
	
	// Collect all transactions for this key
	var allTxs []*model.Transaction
	
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 100).Result()
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
			
			if tx.KeyID == key.ID {
				allTxs = append(allTxs, &tx)
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	// Sort by CreatedAt descending (newest first)
	for i := 0; i < len(allTxs)-1; i++ {
		for j := i + 1; j < len(allTxs); j++ {
			if allTxs[j].CreatedAt.After(allTxs[i].CreatedAt) {
				allTxs[i], allTxs[j] = allTxs[j], allTxs[i]
			}
		}
	}
	
	total := len(allTxs)
	
	// Apply pagination
	if offset >= len(allTxs) {
		return []*model.Transaction{}, total, nil
	}
	
	end := offset + limit
	if end > len(allTxs) {
		end = len(allTxs)
	}
	
	return allTxs[offset:end], total, nil
}

// ==================== User Operations (stored in Redis) ====================

func (s *RedisStore) CreateUser(ctx context.Context, user *model.User) error {
	// Check if email already exists
	exists, err := s.client.Exists(ctx, KeyPrefixUserEmail+user.Email).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return errors.New("email already registered")
	}
	
	// Store user data
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	
	pipe := s.client.Pipeline()
	pipe.Set(ctx, KeyPrefixUser+user.ID, userJSON, 0)
	pipe.Set(ctx, KeyPrefixUserEmail+user.Email, user.ID, 0)
	
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	// Get user ID by email
	userID, err := s.client.Get(ctx, KeyPrefixUserEmail+email).Result()
	if err == redis.Nil {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	
	return s.GetUserByID(ctx, userID)
}

func (s *RedisStore) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	userJSON, err := s.client.Get(ctx, KeyPrefixUser+userID).Result()
	if err == redis.Nil {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	
	var user model.User
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, err
	}
	
	return &user, nil
}
