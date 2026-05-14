// Command: clear one account so the email can register again.
// Usage (from sub2api-go): go run ./cmd/wipe-user you@example.com
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"sub2api-go/internal/config"
	"sub2api-go/internal/store"
)

func main() {
	_ = godotenv.Load()

	email := strings.TrimSpace(os.Getenv("WIPE_USER_EMAIL"))
	if len(os.Args) >= 2 {
		email = strings.TrimSpace(os.Args[1])
	}
	if email == "" {
		fmt.Println("用法: cd sub2api-go && go run ./cmd/wipe-user <邮箱>")
		fmt.Println("  或: WIPE_USER_EMAIL=邮箱 go run ./cmd/wipe-user")
		os.Exit(1)
	}

	ctx := context.Background()
	cfg := config.Load()

	sqliteDB, err := store.NewSQLiteStore("./data/sub2api.db")
	if err != nil {
		log.Printf("sqlite ./data/sub2api.db: %v", err)
		sqliteDB = nil
	}
	if sqliteDB != nil {
		defer func() { _ = sqliteDB.Close() }()
	}

	var linkedUserID string
	redisURL := cfg.RedisURL
	if strings.TrimSpace(redisURL) == "" {
		redisURL = "redis://localhost:6379"
	}

	r, err := store.NewRedisStore(redisURL, sqliteDB)
	if err == nil {
		defer func() { _ = r.Close() }()
		if u, e := r.GetUserByEmail(ctx, email); e == nil {
			linkedUserID = u.ID
		}
		if e := r.DeleteUserByEmail(ctx, email); e != nil {
			log.Fatalf("redis 清理失败: %v", e)
		}
		log.Printf("redis: 已删除用户/Key/验证码相关数据: %s", email)
	} else {
		log.Printf("redis 不可用 (%v)，跳过 Redis 清理（若仅用内存库请重启服务）", err)
	}

	if sqliteDB != nil {
		if err := sqliteDB.DeleteRegistrationByEmail(ctx, email, linkedUserID); err != nil {
			log.Fatalf("sqlite 清理失败: %v", err)
		}
		log.Printf("sqlite: 已清理该邮箱相关 users/api_keys/transactions/register_otps/reset_password_otps: %s", email)
	}

	log.Println("完成，可用同一邮箱重新注册。")
}
