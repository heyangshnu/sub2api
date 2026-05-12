package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

type contextKey string

const (
	KeyAPIKey contextKey = "api_key"
	KeyHash   contextKey = "key_hash"
)

// AuthMiddleware validates API key from Authorization header
func AuthMiddleware(s store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("invalid_request_error", "Missing Authorization header"))
			c.Abort()
			return
		}

		// Support both "Bearer sk-xxx" and "sk-xxx" formats
		rawKey := strings.TrimPrefix(auth, "Bearer ")
		rawKey = strings.TrimSpace(rawKey)

		if rawKey == "" {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("invalid_request_error", "Invalid Authorization header"))
			c.Abort()
			return
		}

		// Validate key
		apiKey, err := s.ValidateKey(c.Request.Context(), rawKey)
		if err != nil {
			switch err {
			case store.ErrKeyNotFound:
				c.JSON(http.StatusUnauthorized, model.NewAPIError("invalid_api_key", "Invalid API key"))
			case store.ErrKeyDisabled:
				c.JSON(http.StatusForbidden, model.NewAPIError("invalid_api_key", "API key is disabled"))
			default:
				c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to validate API key"))
			}
			c.Abort()
			return
		}

		// Store key info in context
		c.Set(string(KeyAPIKey), apiKey)
		c.Set(string(KeyHash), apiKey.KeyHash)
		
		c.Next()
	}
}

// AdminAuthMiddleware validates admin key from X-Admin-Key header
func AdminAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminKey := c.GetHeader("X-Admin-Key")
		if adminKey == "" {
			adminKey = c.GetHeader("Authorization")
			adminKey = strings.TrimPrefix(adminKey, "Bearer ")
		}

		if adminKey != cfg.AdminKey {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("invalid_request_error", "Invalid admin key"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAPIKey retrieves the API key from context
func GetAPIKey(c *gin.Context) *model.APIKey {
	if key, exists := c.Get(string(KeyAPIKey)); exists {
		return key.(*model.APIKey)
	}
	return nil
}

// GetKeyHash retrieves the key hash from context
func GetKeyHash(c *gin.Context) string {
	if hash, exists := c.Get(string(KeyHash)); exists {
		return hash.(string)
	}
	return ""
}

// CORSMiddleware handles CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	// Simple implementation; use UUID in production
	return "req_" + randomString(16)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

// WithAPIKey adds API key info to context
func WithAPIKey(ctx context.Context, key *model.APIKey) context.Context {
	return context.WithValue(ctx, KeyAPIKey, key)
}
