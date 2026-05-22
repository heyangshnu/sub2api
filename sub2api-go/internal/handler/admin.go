package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

type AdminHandler struct {
	store store.Store
}

func NewAdminHandler(s store.Store) *AdminHandler {
	return &AdminHandler{store: s}
}

// CreateKey handles POST /admin/keys
func (h *AdminHandler) CreateKey(c *gin.Context) {
	var req model.CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	rawKey, apiKey, err := h.store.CreateKey(c.Request.Context(), req.UserID, req.Name, req.Balance, req.RateLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to create key"))
		return
	}

	c.JSON(http.StatusCreated, model.CreateKeyResponse{
		Key:     rawKey,
		KeyID:   apiKey.ID,
		UserID:  apiKey.UserID,
		Balance: apiKey.Balance,
	})
}

// ListKeys handles GET /admin/keys
func (h *AdminHandler) ListKeys(c *gin.Context) {
	userID := c.Query("user_id")

	keys, err := h.store.ListKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to list keys"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// GetKey handles GET /admin/keys/:id
func (h *AdminHandler) GetKey(c *gin.Context) {
	keyID := c.Param("id")

	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if err == store.ErrKeyNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "Key not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get key"))
		}
		return
	}

	c.JSON(http.StatusOK, key)
}

// TopupKey handles POST /admin/keys/:id/topup
func (h *AdminHandler) TopupKey(c *gin.Context) {
	keyID := c.Param("id")

	var req model.TopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request: "+err.Error()))
		return
	}

	// Get key first to get the hash
	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if err == store.ErrKeyNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "Key not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get key"))
		}
		return
	}

	if err := h.store.Topup(c.Request.Context(), key.KeyHash, req.Amount, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to topup"))
		return
	}

	// Get updated key
	key, _ = h.store.GetKeyByID(c.Request.Context(), keyID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Topup successful",
		"balance": key.Balance,
	})
}

// ==================== User Endpoints ====================

type UserHandler struct {
	store store.Store
	cfg   *config.Config
}

func NewUserHandler(s store.Store, cfg *config.Config) *UserHandler {
	return &UserHandler{store: s, cfg: cfg}
}

// GetUsage handles GET /v1/usage
func (h *UserHandler) GetUsage(c *gin.Context) {
	keyHash := middleware.GetKeyHash(c)

	usage, err := h.store.GetUsageStats(c.Request.Context(), keyHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get usage"))
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetTransactions handles GET /v1/transactions
func (h *UserHandler) GetTransactions(c *gin.Context) {
	keyHash := middleware.GetKeyHash(c)

	// Parse pagination params
	limit := 20
	offset := 0
	
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	transactions, total, err := h.store.ListTransactions(c.Request.Context(), keyHash, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get transactions"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetModels handles GET /v1/models
func (h *UserHandler) GetModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   ModelsFromConfig(h.cfg),
	})
}

// ==================== Health Endpoint ====================

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"store":  "memory",
	})
}

func HealthHandlerWithStore(c *gin.Context, storeType string) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"store":  storeType,
	})
}
