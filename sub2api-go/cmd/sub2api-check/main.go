// sub2api-check: offline consistency helper — validates one API key against store (Redis).
//
// Usage (from sub2api-go):
//
//	go run ./cmd/sub2api-check -key sk-sub2api-...
//
// Requires REDIS_URL (or default redis://localhost:6379) and optional SQLite at ./data/sub2api.db.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"sub2api-go/internal/config"
	"sub2api-go/internal/store"
)

func main() {
	_ = godotenv.Load()
	raw := flag.String("key", "", "Raw API key (sk-sub2api-...)")
	flag.Parse()
	if *raw == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/sub2api-check -key <raw_api_key>")
		os.Exit(2)
	}

	cfg := config.Load()
	ctx := context.Background()

	var sqlite *store.SQLiteStore
	if s, err := store.NewSQLiteStore("./data/sub2api.db"); err == nil {
		sqlite = s
		defer func() { _ = sqlite.Close() }()
	}

	r, err := store.NewRedisStore(cfg.RedisURL, sqlite)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer func() { _ = r.Close() }()

	key, err := r.ValidateKey(ctx, *raw)
	if err != nil {
		log.Fatalf("validate key: %v", err)
	}

	bal, err := r.GetBalance(ctx, key.KeyHash)
	if err != nil {
		log.Fatalf("balance: %v", err)
	}

	const scanLimit = 5000
	txs, total, err := r.ListTransactions(ctx, key.KeyHash, scanLimit, 0)
	if err != nil {
		log.Fatalf("transactions: %v", err)
	}

	var sumConsume, sumTopup float64
	for _, tx := range txs {
		switch tx.Type {
		case "consume":
			sumConsume += tx.Amount
		case "topup":
			sumTopup += tx.Amount
		}
	}

	fmt.Printf("key_id:        %s\n", key.ID)
	fmt.Printf("key_prefix:    %s\n", key.KeyPrefix)
	fmt.Printf("balance_now:   %.6f\n", bal)
	fmt.Printf("tx_rows_seen:  %d (capped fetch limit %d, total reported %d)\n", len(txs), scanLimit, total)
	fmt.Printf("sum_consume:   %.6f\n", sumConsume)
	fmt.Printf("sum_topup:     %.6f\n", sumTopup)
	fmt.Println()
	fmt.Println("Note: Redis transaction keys may use TTL; older rows can disappear from this scan.")
	fmt.Println("Use this as a sanity check, not a legal ledger audit.")
}
