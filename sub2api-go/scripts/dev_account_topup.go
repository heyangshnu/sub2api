// 本地开发：给指定用户账户充值（USD）
//
//	go run ./scripts/dev_account_topup.go -email heyangshnu@gmail.com -amount 1
//	go run ./scripts/dev_account_topup.go -user user_xxx -amount 1 -paid
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
	cfg := config.Load()

	email := flag.String("email", "", "user email")
	userID := flag.String("user", "", "user id")
	amount := flag.Float64("amount", 1, "USD amount to add")
	setPaid := flag.Bool("paid", true, "set has_paid=true (unlock API key creation)")
	flag.Parse()

	if *email == "" && *userID == "" {
		log.Fatal("usage: -email <email> or -user <user_id> -amount 1")
	}
	if *amount <= 0 {
		log.Fatal("amount must be positive")
	}

	sqliteStore, _ := store.NewSQLiteStore("./data/sub2api.db")
	redisStore, err := store.NewRedisStore(cfg.RedisURL, sqliteStore)
	if err != nil {
		log.Fatalf("redis: %v (is redis-server running?)", err)
	}

	ctx := context.Background()
	uid := *userID
	if uid == "" {
		u, err := redisStore.GetUserByEmail(ctx, *email)
		if err != nil {
			log.Fatalf("user not found for email %q: %v", *email, err)
		}
		uid = u.ID
	}

	if err := redisStore.AccountTopup(ctx, uid, *amount, "admin_topup", "dev script topup", "", *setPaid); err != nil {
		log.Fatalf("topup failed: %v", err)
	}

	bal, _ := redisStore.GetAccountBalance(ctx, uid)
	u, _ := redisStore.GetUserByID(ctx, uid)
	fmt.Fprintf(os.Stdout, "OK user=%s email=%s balance=%.4f has_paid=%v\n", uid, u.Email, bal, u.HasPaid)
}
