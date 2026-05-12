package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

type AuthHandler struct {
	store      store.Store
	jwtSecret  []byte
	inviteCode string
}

func NewAuthHandler(s store.Store, jwtSecret, inviteCode string) *AuthHandler {
	return &AuthHandler{
		store:      s,
		jwtSecret:  []byte(jwtSecret),
		inviteCode: inviteCode,
	}
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	// 验证邀请码（如果配置了邀请码）
	if h.inviteCode != "" && req.InviteCode != h.inviteCode {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "Invalid invite code"))
		return
	}

	// Check if user exists
	_, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, model.NewAPIError("conflict", "Email already registered"))
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to hash password"))
		return
	}

	// Create user
	now := time.Now()
	userID := generateUserID(req.Email)
	user := &model.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Name:         req.Name,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if user.Name == "" {
		user.Name = req.Email
	}

	if err := h.store.CreateUser(c.Request.Context(), user); err != nil {
		if err == store.ErrUserExists {
			c.JSON(http.StatusConflict, model.NewAPIError("conflict", "Email already registered"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to create user"))
		return
	}

	// Create initial API key for the user (with $0 balance)
	rawKey, _, err := h.store.CreateKey(c.Request.Context(), userID, "Default Key", 0, 60)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to create API key"))
		return
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate token"))
		return
	}

	c.JSON(http.StatusCreated, model.AuthResponse{
		Token:  token,
		User:   user,
		APIKey: rawKey,
	})
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	// Get user
	user, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid email or password"))
		return
	}

	// Check status
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "Account is disabled"))
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid email or password"))
		return
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{
		Token: token,
		User:  user,
	})
}

// GetMe handles GET /auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Not authenticated"))
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
		return
	}

	c.JSON(http.StatusOK, user)
}

// JWTAuthMiddleware validates JWT token
func (h *AuthHandler) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Missing authorization header"))
			c.Abort()
			return
		}

		// Remove "Bearer " prefix
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return h.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid token"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid token claims"))
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Invalid user ID in token"))
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func (h *AuthHandler) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func generateUserID(email string) string {
	h := sha256.Sum256([]byte(email + time.Now().String()))
	return "user_" + hex.EncodeToString(h[:8])
}
