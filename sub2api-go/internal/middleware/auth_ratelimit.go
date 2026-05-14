package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"sub2api-go/internal/model"
)

// AuthRateLimitMiddleware limits brute-force on public /auth/* POST endpoints (per client IP).
// Requires Redis; if client is nil, it is a no-op (local dev without Redis).
func AuthRateLimitMiddleware(client *redis.Client) gin.HandlerFunc {
	if client == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		// GET /auth/config stays unlimited
		if path == "/auth/config" {
			c.Next()
			return
		}

		var window time.Duration
		var max int64
		var bucket string
		switch path {
		case "/auth/login":
			bucket, window, max = "login", time.Minute, 30
		case "/auth/register":
			bucket, window, max = "register", time.Minute, 12
		case "/auth/send-register-code":
			bucket, window, max = "regcode", time.Hour, 20
		case "/auth/send-reset-password-code":
			bucket, window, max = "rstcode", time.Hour, 20
		case "/auth/reset-password":
			bucket, window, max = "resetpw", time.Minute, 30
		default:
			c.Next()
			return
		}

		ip := c.ClientIP()
		now := time.Now().UTC()
		var winKey string
		if window >= time.Hour {
			winKey = now.Format("2006010215") // hourly bucket for long windows
		} else {
			winKey = now.Format("200601021504") // per minute
		}
		rlKey := fmt.Sprintf("authrl:%s:%s:%s", bucket, ip, winKey)

		ctx := c.Request.Context()
		count, err := client.Incr(ctx, rlKey).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			ttl := window + 10*time.Second
			_ = client.Expire(ctx, rlKey, ttl).Err()
		}
		if count > max {
			c.Header("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
			c.JSON(http.StatusTooManyRequests, model.NewAPIError("rate_limit_error",
				"Too many attempts; please try again later"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// StripBearerPrefix returns the token part after optional "Bearer " prefix (case-sensitive per HTTP).
func StripBearerPrefix(auth string) string {
	auth = strings.TrimSpace(auth)
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return auth
}
