package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"sub2api-go/internal/model"
)

// DefaultRateLimit 默认频次限制：60 次/分钟
const DefaultRateLimit = 60

// RateLimitMiddleware 基于 Redis 的每 Key 频次限制
// 同时校验 IP 白名单
func RateLimitMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKey(c)
		if apiKey == nil {
			c.Next()
			return
		}

		// ========== IP 白名单校验 ==========
		if len(apiKey.IPWhitelist) > 0 {
			clientIP := getClientIP(c)
			allowed := false
			for _, ip := range apiKey.IPWhitelist {
				ip = strings.TrimSpace(ip)
				if ip == "" {
					continue
				}
				// 支持 CIDR
				if strings.Contains(ip, "/") {
					_, ipNet, err := net.ParseCIDR(ip)
					if err == nil && ipNet.Contains(net.ParseIP(clientIP)) {
						allowed = true
						break
					}
				} else if ip == clientIP {
					allowed = true
					break
				}
			}
			if !allowed {
				c.JSON(http.StatusForbidden, model.NewAPIError("ip_not_allowed",
					fmt.Sprintf("IP %s is not in the whitelist", clientIP)))
				c.Abort()
				return
			}
		}

		// ========== 频次限制校验 ==========
		if redisClient == nil {
			c.Next()
			return
		}

		limit := apiKey.RateLimit
		if limit <= 0 {
			limit = DefaultRateLimit
		}

		// 使用滑动窗口：当前分钟的键
		now := time.Now()
		window := now.Format("200601021504") // 分钟粒度
		rlKey := fmt.Sprintf("ratelimit:%s:%s", apiKey.KeyHash, window)

		ctx := c.Request.Context()
		count, err := redisClient.Incr(ctx, rlKey).Result()
		if err != nil {
			// Redis 出错时放行，不影响服务
			c.Next()
			return
		}
		if count == 1 {
			// 第一次请求设置过期时间
			redisClient.Expire(ctx, rlKey, 70*time.Second)
		}

		if count > int64(limit) {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, model.NewAPIError("rate_limit_exceeded",
				fmt.Sprintf("Rate limit exceeded: %d requests per minute", limit)))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(limit)-count))
		c.Next()
	}
}

// getClientIP 获取真实客户端 IP（考虑 Nginx 代理）
func getClientIP(c *gin.Context) string {
	// 优先从 X-Real-IP 获取（Nginx 设置）
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	// 从 X-Forwarded-For 取第一个
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return c.ClientIP()
}
