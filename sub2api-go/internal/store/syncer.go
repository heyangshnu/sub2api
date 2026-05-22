package store

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"sub2api-go/internal/model"
)

// Syncer handles Redis → SQLite synchronization
type Syncer struct {
	redis    *RedisStore
	sqlite   *SQLiteStore
	interval time.Duration
	stopCh   chan struct{}
}

func NewSyncer(redis *RedisStore, sqlite *SQLiteStore, interval time.Duration) *Syncer {
	return &Syncer{
		redis:    redis,
		sqlite:   sqlite,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync
func (s *Syncer) Start() {
	go s.syncLoop()
	log.Printf("[Syncer] Started with interval %v", s.interval)
}

// Stop halts the sync loop
func (s *Syncer) Stop() {
	close(s.stopCh)
	log.Println("[Syncer] Stopped")
}

func (s *Syncer) syncLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.SyncAll(context.Background()); err != nil {
				log.Printf("[Syncer] Sync error: %v", err)
			}
		case <-s.stopCh:
			return
		}
	}
}

// SyncAll synchronizes all data from Redis to SQLite
func (s *Syncer) SyncAll(ctx context.Context) error {
	start := time.Now()
	
	// Sync API keys
	keysCount, err := s.syncKeys(ctx)
	if err != nil {
		log.Printf("[Syncer] Keys sync error: %v", err)
	}

	// Sync transactions
	txCount, err := s.syncTransactions(ctx)
	if err != nil {
		log.Printf("[Syncer] Transactions sync error: %v", err)
	}

	userCount, err := s.syncUsers(ctx)
	if err != nil {
		log.Printf("[Syncer] Users sync error: %v", err)
	}

	outboxCount, _ := s.ProcessOutbox(ctx)

	log.Printf("[Syncer] Sync completed: %d keys, %d ledger, %d users, %d outbox in %v",
		keysCount, txCount, userCount, outboxCount, time.Since(start))

	return nil
}

func (s *Syncer) syncKeys(ctx context.Context) (int, error) {
	// Get all keys from Redis
	keys, err := s.redis.ListKeys(ctx, "")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, key := range keys {
		if err := s.sqlite.SaveKey(ctx, key); err != nil {
			log.Printf("[Syncer] Failed to save key %s: %v", key.ID, err)
			continue
		}
		count++
	}

	return count, nil
}

func (s *Syncer) syncTransactions(ctx context.Context) (int, error) {
	// Scan all transaction keys from Redis
	client := s.redis.Client()
	
	var cursor uint64
	count := 0
	
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 100).Result()
		if err != nil {
			return count, err
		}

		for _, k := range keys {
			txJSON, err := client.Get(ctx, k).Result()
			if err != nil {
				continue
			}

			var tx model.Transaction
			if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
				continue
			}

			// Save to SQLite
			if err := s.sqlite.SaveTransaction(ctx, &tx); err != nil {
				// Ignore duplicate key errors (transaction already synced)
				continue
			}

			// Delete from Redis after successful sync (optional)
			// client.Del(ctx, k)
			
			count++
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

func (s *Syncer) syncUsers(ctx context.Context) (int, error) {
	if s.sqlite == nil {
		return 0, nil
	}
	var cursor uint64
	count := 0
	for {
		keys, next, err := s.redis.client.Scan(ctx, cursor, KeyPrefixUser+"*", 100).Result()
		if err != nil {
			return count, err
		}
		for _, k := range keys {
			userID := strings.TrimPrefix(k, KeyPrefixUser)
			user, err := s.redis.GetUserByID(ctx, userID)
			if err != nil {
				continue
			}
			_ = s.sqlite.CreateUser(ctx, user)
			s.redis.writeThroughUserAccount(ctx, user)
			count++
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return count, nil
}

// SyncKeyBalance syncs a specific key's balance from Redis to SQLite
func (s *Syncer) SyncKeyBalance(ctx context.Context, keyHash string) error {
	balance, err := s.redis.GetBalance(ctx, keyHash)
	if err != nil {
		return err
	}

	key, err := s.redis.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return err
	}

	key.Balance = balance
	return s.sqlite.SaveKey(ctx, key)
}
