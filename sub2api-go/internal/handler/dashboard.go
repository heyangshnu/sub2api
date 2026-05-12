package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

// DashboardHandler 处理用户 Dashboard 的 Key 管理
type DashboardHandler struct {
	store store.Store
}

func NewDashboardHandler(s store.Store) *DashboardHandler {
	return &DashboardHandler{store: s}
}

// CreateKey 用户从 Dashboard 创建 Key（需二次验证密码）
// POST /dashboard/keys
func (h *DashboardHandler) CreateKey(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	var req model.CreateUserKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	// 二次验证密码
	user, err := h.store.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "Password verification failed"))
		return
	}

	// 创建 Key（初始余额 0，用户充值后使用）
	rateLimit := req.RateLimit
	if rateLimit <= 0 {
		rateLimit = 60 // 默认 60/分钟
	}
	name := req.Name
	if name == "" {
		name = "Dashboard Key"
	}

	rawKey, apiKey, err := h.store.CreateKey(c.Request.Context(), uid, name, 0, rateLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to create key"))
		return
	}

	// 明文 Key 仅此一次返回
	c.JSON(http.StatusCreated, gin.H{
		"key":        rawKey,
		"key_id":     apiKey.ID,
		"key_prefix": apiKey.KeyPrefix,
		"name":       apiKey.Name,
		"balance":    apiKey.Balance,
		"rate_limit": apiKey.RateLimit,
		"warning":    "This is the only time you will see this key. Please save it securely.",
	})
}

// UpdateKeySettings 更新 Key 设置（IP 白名单 / 频次）
// PATCH /dashboard/keys/:id
func (h *DashboardHandler) UpdateKeySettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	keyID := c.Param("id")

	var req model.UpdateKeySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	// 校验 Key 属于当前用户
	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "Key not found"))
		return
	}
	if key.UserID != uid {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "You don't own this key"))
		return
	}

	// 频次限制合法性：1-3600/分钟
	if req.RateLimit > 0 && (req.RateLimit < 1 || req.RateLimit > 3600) {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "rate_limit must be between 1 and 3600"))
		return
	}

	if err := h.store.UpdateKeySettings(c.Request.Context(), key.KeyHash, req.IPWhitelist, req.RateLimit); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to update settings"))
		return
	}

	// 返回更新后的 Key
	updatedKey, _ := h.store.GetKeyByID(c.Request.Context(), keyID)
	c.JSON(http.StatusOK, updatedKey)
}

// DeleteKey 删除 Key
// DELETE /dashboard/keys/:id
func (h *DashboardHandler) DeleteKey(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	keyID := c.Param("id")

	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "Key not found"))
		return
	}
	if key.UserID != uid {
		c.JSON(http.StatusForbidden, model.NewAPIError("forbidden", "You don't own this key"))
		return
	}

	if err := h.store.DeleteKey(c.Request.Context(), key.KeyHash); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to delete key"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Key deleted successfully"})
}
