package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

func parseUsageDays(c *gin.Context, def int) int {
	days := def
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	return days
}

// GetUsageDaily GET /dashboard/usage-daily?scope=account&days=14
// or GET /dashboard/usage-daily?key_id=&days=14 — per-key chart.
func (h *DashboardHandler) GetUsageDaily(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	days := parseUsageDays(c, 14)

	if strings.EqualFold(c.Query("scope"), "account") {
		points, err := h.store.AggregateUserConsumeByDay(c.Request.Context(), uid, days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate usage"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"scope":  "account",
			"days":   days,
			"points": points,
		})
		return
	}

	keyID := c.Query("key_id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id is required (or use scope=account)"})
		return
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

// GetUsageSummary GET /dashboard/usage/summary
func (h *DashboardHandler) GetUsageSummary(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	summary, err := h.store.GetUsageSummary(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load usage summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetUsageByModel GET /dashboard/usage/by-model?days=30
func (h *DashboardHandler) GetUsageByModel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	days := parseUsageDays(c, 30)
	rows, err := h.store.AggregateConsumeByModel(c.Request.Context(), uid, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate by model"})
		return
	}
	if rows == nil {
		rows = []model.ModelUsageRow{}
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "rows": rows})
}

// ExportUsage GET /dashboard/usage/export?month=YYYY-MM
func (h *DashboardHandler) ExportUsage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	csv, err := h.store.ExportUsageCSV(c.Request.Context(), uid, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export usage"})
		return
	}
	filename := "usage-" + month + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csv)
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
