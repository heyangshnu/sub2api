package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"sub2api-go/internal/model"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// MemoryStore is an in-memory implementation of the store interface.
// Used for MVP phase; will be replaced with Redis + SQLite later.
type MemoryStore struct {
	mu           sync.RWMutex
	keys         map[string]*model.APIKey    // keyHash -> APIKey
	users        map[string]*model.User      // email -> User
	usersById    map[string]*model.User      // id -> User
	registerOTP        map[string]*memRegisterOTP // normalized email -> pending registration code
	resetPasswordOTP   map[string]*memRegisterOTP // normalized email -> pending reset code
	requestLogs        map[string][]*model.RequestLogEntry // keyID -> newest first
	transactions       []model.Transaction
	keyCounter   int
}

type memRegisterOTP struct {
	hash      string
	expiresAt time.Time
	createdAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		keys:         make(map[string]*model.APIKey),
		users:        make(map[string]*model.User),
		usersById:    make(map[string]*model.User),
		registerOTP:       make(map[string]*memRegisterOTP),
		resetPasswordOTP:  make(map[string]*memRegisterOTP),
		requestLogs:       make(map[string][]*model.RequestLogEntry),
		transactions: make([]model.Transaction, 0),
	}
}

// HashKey returns SHA256 hash of the API key
func HashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// GenerateAPIKey creates a new API key with the format sk-sub2api-<random>
func (s *MemoryStore) GenerateAPIKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyCounter++
	
	// Generate a simple key for MVP; use crypto/rand in production
	timestamp := time.Now().UnixNano()
	data := []byte{byte(s.keyCounter), byte(timestamp >> 8), byte(timestamp >> 16), byte(timestamp >> 24)}
	h := sha256.Sum256(data)
	random := hex.EncodeToString(h[:16])
	
	return "sk-sub2api-" + random
}

// CreateKey creates a new API key
func (s *MemoryStore) CreateKey(ctx context.Context, userID, name string, balance float64, rateLimit int) (string, *model.APIKey, error) {
	rawKey := s.GenerateAPIKey()
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
	
	s.mu.Lock()
	s.keys[keyHash] = key
	s.mu.Unlock()
	
	return rawKey, key, nil
}

// ValidateKey validates an API key and returns the associated key record
func (s *MemoryStore) ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := HashKey(rawKey)
	
	s.mu.RLock()
	key, exists := s.keys[keyHash]
	s.mu.RUnlock()
	
	if !exists {
		return nil, ErrKeyNotFound
	}
	
	if key.Status != "active" {
		return nil, ErrKeyDisabled
	}
	
	return key, nil
}

// GetBalance returns the current balance for a key
func (s *MemoryStore) GetBalance(ctx context.Context, keyHash string) (float64, error) {
	s.mu.RLock()
	key, exists := s.keys[keyHash]
	s.mu.RUnlock()
	
	if !exists {
		return 0, ErrKeyNotFound
	}
	
	return key.Balance, nil
}

// PreDeduct attempts to pre-deduct an estimated amount
// Returns error if insufficient balance
func (s *MemoryStore) PreDeduct(ctx context.Context, keyHash string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	
	if key.Balance < amount {
		return ErrInsufficientBalance
	}
	
	key.Balance -= amount
	key.UpdatedAt = time.Now()
	
	return nil
}

// FinalizeDeduct adjusts the balance based on actual usage
// If actualAmount < preDeducted, refunds the difference
// If actualAmount > preDeducted, deducts additional amount
func (s *MemoryStore) FinalizeDeduct(ctx context.Context, keyHash string, preDeducted, actualAmount float64, usage model.Usage, modelName, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	
	diff := actualAmount - preDeducted
	balanceBefore := key.Balance
	
	// Adjust balance
	key.Balance -= diff
	now := time.Now()
	key.UpdatedAt = now
	key.LastUsedAt = &now
	
	// Record transaction
	tx := model.Transaction{
		ID:            generateTxID(),
		KeyID:         key.ID,
		Type:          "consume",
		Amount:        actualAmount,
		BalanceBefore: balanceBefore + preDeducted, // Balance before any deduction
		BalanceAfter:  key.Balance,
		Model:         modelName,
		InputTokens:   usage.PromptTokens,
		OutputTokens:  usage.CompletionTokens,
		RequestID:     requestID,
		CreatedAt:     now,
	}
	s.transactions = append(s.transactions, tx)
	
	return nil
}

// RefundPreDeduct refunds a pre-deducted amount (used when request fails)
func (s *MemoryStore) RefundPreDeduct(ctx context.Context, keyHash string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	
	key.Balance += amount
	key.UpdatedAt = time.Now()
	
	return nil
}

// Topup adds balance to a key
func (s *MemoryStore) Topup(ctx context.Context, keyHash string, amount float64, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	
	balanceBefore := key.Balance
	key.Balance += amount
	now := time.Now()
	key.UpdatedAt = now
	
	// Record transaction
	tx := model.Transaction{
		ID:            generateTxID(),
		KeyID:         key.ID,
		Type:          "topup",
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  key.Balance,
		CreatedAt:     now,
	}
	s.transactions = append(s.transactions, tx)
	
	return nil
}

// GetKeyByHash returns a key by its hash
func (s *MemoryStore) GetKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	s.mu.RLock()
	key, exists := s.keys[keyHash]
	s.mu.RUnlock()
	
	if !exists {
		return nil, ErrKeyNotFound
	}
	
	return key, nil
}

// GetKeyByID returns a key by its ID
func (s *MemoryStore) GetKeyByID(ctx context.Context, keyID string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, key := range s.keys {
		if key.ID == keyID {
			return key, nil
		}
	}
	
	return nil, ErrKeyNotFound
}

// ListKeys returns all keys for a user
func (s *MemoryStore) ListKeys(ctx context.Context, userID string) ([]*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var keys []*model.APIKey
	for _, key := range s.keys {
		if userID == "" || key.UserID == userID {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

func (s *MemoryStore) UpdateKeySettings(ctx context.Context, keyHash string, ipWhitelist []string, rateLimit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	
	if ipWhitelist != nil {
		key.IPWhitelist = ipWhitelist
	}
	if rateLimit > 0 {
		key.RateLimit = rateLimit
	}
	key.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) DeleteKey(ctx context.Context, keyHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[keyHash]
	if !exists {
		return ErrKeyNotFound
	}
	delete(s.requestLogs, key.ID)
	delete(s.keys, keyHash)
	return nil
}

// GetUsageStats returns usage statistics for a key
func (s *MemoryStore) GetUsageStats(ctx context.Context, keyHash string) (*model.UsageResponse, error) {
	s.mu.RLock()
	key, exists := s.keys[keyHash]
	if !exists {
		s.mu.RUnlock()
		return nil, ErrKeyNotFound
	}
	
	var totalUsed float64
	var requestCount int
	var lastRequest time.Time
	
	for _, tx := range s.transactions {
		if tx.KeyID == key.ID && tx.Type == "consume" {
			totalUsed += tx.Amount
			requestCount++
			if tx.CreatedAt.After(lastRequest) {
				lastRequest = tx.CreatedAt
			}
		}
	}
	s.mu.RUnlock()
	
	resp := &model.UsageResponse{
		Balance:      key.Balance,
		TotalUsed:    totalUsed,
		RequestCount: requestCount,
	}
	
	if !lastRequest.IsZero() {
		resp.LastRequestAt = lastRequest.Format(time.RFC3339)
	}
	
	return resp, nil
}

// ListTransactions returns transactions for a key with pagination
func (s *MemoryStore) ListTransactions(ctx context.Context, keyHash string, limit, offset int) ([]*model.Transaction, int, error) {
	s.mu.RLock()
	key, exists := s.keys[keyHash]
	if !exists {
		s.mu.RUnlock()
		return nil, 0, ErrKeyNotFound
	}
	
	// Filter transactions for this key
	var keyTxs []*model.Transaction
	for i := range s.transactions {
		if s.transactions[i].KeyID == key.ID {
			keyTxs = append(keyTxs, &s.transactions[i])
		}
	}
	s.mu.RUnlock()
	
	// Sort by CreatedAt descending (newest first)
	for i := 0; i < len(keyTxs)-1; i++ {
		for j := i + 1; j < len(keyTxs); j++ {
			if keyTxs[j].CreatedAt.After(keyTxs[i].CreatedAt) {
				keyTxs[i], keyTxs[j] = keyTxs[j], keyTxs[i]
			}
		}
	}
	
	total := len(keyTxs)
	
	// Apply pagination
	if offset >= len(keyTxs) {
		return []*model.Transaction{}, total, nil
	}
	
	end := offset + limit
	if end > len(keyTxs) {
		end = len(keyTxs)
	}
	
	return keyTxs[offset:end], total, nil
}

func generateTxID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return hex.EncodeToString(h[:8])
}

// ==================== User Operations ====================

func (s *MemoryStore) CreateUser(ctx context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Email]; exists {
		return ErrUserExists
	}

	s.users[user.Email] = user
	s.usersById[user.ID] = user
	return nil
}

func (s *MemoryStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.usersById[userID]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) UpdateUser(ctx context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersById[user.ID]; !exists {
		return ErrUserNotFound
	}
	s.users[user.Email] = user
	s.usersById[user.ID] = user
	return nil
}

func normalizeRegisterEmailMem(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *MemoryStore) SaveRegisterOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error {
	em := normalizeRegisterEmailMem(email)
	s.mu.Lock()
	defer s.mu.Unlock()

	if prev, ok := s.registerOTP[em]; ok {
		if time.Since(prev.createdAt) < 60*time.Second {
			return ErrRegisterOTPCooldown
		}
	}
	s.registerOTP[em] = &memRegisterOTP{
		hash:      codeHash,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}
	return nil
}

func (s *MemoryStore) ConsumeRegisterOTP(ctx context.Context, email, plainCode string) error {
	em := normalizeRegisterEmailMem(email)
	s.mu.Lock()
	defer s.mu.Unlock()

	ent, ok := s.registerOTP[em]
	if !ok {
		return ErrRegisterOTPInvalid
	}
	if time.Now().After(ent.expiresAt) {
		delete(s.registerOTP, em)
		return ErrRegisterOTPInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(ent.hash), []byte(strings.TrimSpace(plainCode))); err != nil {
		return ErrRegisterOTPInvalid
	}
	delete(s.registerOTP, em)
	return nil
}

func (s *MemoryStore) SaveResetPasswordOTP(ctx context.Context, email, codeHash string, expiresAt, createdAt time.Time) error {
	em := normalizeRegisterEmailMem(email)
	s.mu.Lock()
	defer s.mu.Unlock()

	if prev, ok := s.resetPasswordOTP[em]; ok {
		if time.Since(prev.createdAt) < 60*time.Second {
			return ErrResetPasswordOTPCooldown
		}
	}
	s.resetPasswordOTP[em] = &memRegisterOTP{
		hash:      codeHash,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}
	return nil
}

func (s *MemoryStore) ConsumeResetPasswordOTP(ctx context.Context, email, plainCode string) error {
	em := normalizeRegisterEmailMem(email)
	s.mu.Lock()
	defer s.mu.Unlock()

	ent, ok := s.resetPasswordOTP[em]
	if !ok {
		return ErrResetPasswordOTPInvalid
	}
	if time.Now().After(ent.expiresAt) {
		delete(s.resetPasswordOTP, em)
		return ErrResetPasswordOTPInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(ent.hash), []byte(strings.TrimSpace(plainCode))); err != nil {
		return ErrResetPasswordOTPInvalid
	}
	delete(s.resetPasswordOTP, em)
	return nil
}

const memMaxRequestLogs = 200

func (s *MemoryStore) AggregateConsumeByDay(ctx context.Context, keyHash string, days int) ([]model.DailyUsagePoint, error) {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	s.mu.RLock()
	key, ok := s.keys[keyHash]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrKeyNotFound
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	byDay := make(map[string]*model.DailyUsagePoint)
	for _, tx := range s.transactions {
		if tx.KeyID == key.ID && tx.Type == "consume" && !tx.CreatedAt.UTC().Before(cutoff) {
			d := tx.CreatedAt.UTC().Format("2006-01-02")
			if byDay[d] == nil {
				byDay[d] = &model.DailyUsagePoint{Date: d}
			}
			byDay[d].TotalConsumed += tx.Amount
			byDay[d].RequestCount++
		}
	}
	s.mu.RUnlock()
	dates := make([]string, 0, len(byDay))
	for d := range byDay {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]model.DailyUsagePoint, 0, len(dates))
	for _, d := range dates {
		out = append(out, *byDay[d])
	}
	return out, nil
}

func (s *MemoryStore) AppendRequestLog(ctx context.Context, entry *model.RequestLogEntry) error {
	if entry == nil || entry.KeyID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = generateTxID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	cp := *entry
	list := s.requestLogs[entry.KeyID]
	next := append([]*model.RequestLogEntry{&cp}, list...)
	if len(next) > memMaxRequestLogs {
		next = next[:memMaxRequestLogs]
	}
	s.requestLogs[entry.KeyID] = next
	return nil
}

func (s *MemoryStore) ListRequestLogs(ctx context.Context, keyID string, limit, offset int) ([]*model.RequestLogEntry, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.requestLogs[keyID]
	total := len(list)
	if offset >= total {
		return []*model.RequestLogEntry{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	slice := list[offset:end]
	out := make([]*model.RequestLogEntry, 0, len(slice))
	for _, e := range slice {
		if e == nil {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, total, nil
}

// DeleteUserByEmail removes user, keys, txs, and registration / reset-password OTP (memory store only).
func (s *MemoryStore) DeleteUserByEmail(ctx context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.registerOTP, normalizeRegisterEmailMem(email))
	delete(s.resetPasswordOTP, normalizeRegisterEmailMem(email))

	u, ok := s.users[email]
	if !ok {
		return nil
	}
	userID := u.ID

	keyIDs := make(map[string]struct{})
	for kh, k := range s.keys {
		if k != nil && k.UserID == userID {
			keyIDs[k.ID] = struct{}{}
			delete(s.requestLogs, k.ID)
			delete(s.keys, kh)
		}
	}

	out := s.transactions[:0]
	for _, tx := range s.transactions {
		if _, drop := keyIDs[tx.KeyID]; !drop {
			out = append(out, tx)
		}
	}
	s.transactions = out

	delete(s.users, email)
	delete(s.usersById, userID)
	return nil
}
