package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/store"
)

// GetUsageDaily GET /dashboard/usage-daily?key_id=&days=14 — aggregates consume transactions by UTC day for one owned key.
func (h *DashboardHandler) GetUsageDaily(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	keyID := c.Query("key_id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id is required"})
		return
	}
	days := 14
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}

	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
		return
	}
	if key.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this key"})
		return
	}

	points, err := h.store.AggregateConsumeByDay(c.Request.Context(), key.KeyHash, days)
	if err != nil {
		if err == store.ErrKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key_id": keyID,
		"days":   days,
		"points": points,
	})
}

// ListRequestLogs GET /dashboard/request-logs?key_id=&limit=20&offset=0
func (h *DashboardHandler) ListRequestLogs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	keyID := c.Query("key_id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id is required"})
		return
	}
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	key, err := h.store.GetKeyByID(c.Request.Context(), keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
		return
	}
	if key.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this key"})
		return
	}

	logs, total, err := h.store.ListRequestLogs(c.Request.Context(), keyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
