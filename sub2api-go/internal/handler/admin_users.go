package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

// ListUsers GET /admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit, offset := adminPagination(c)
	users, total, err := h.store.AdminListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		if err == store.ErrAdminNotSupported {
			c.JSON(http.StatusNotImplemented, model.NewAPIError("not_supported", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to list users"))
		return
	}
	if users == nil {
		users = []*model.User{}
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": total, "limit": limit, "offset": offset})
}

// GetUser GET /admin/users/:id
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	user, err := h.store.AdminGetUser(c.Request.Context(), userID)
	if err != nil {
		if err == store.ErrUserNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get user"))
		return
	}
	c.JSON(http.StatusOK, user)
}

// AdjustUserBalance PATCH /admin/users/:id/balance
func (h *AdminHandler) AdjustUserBalance(c *gin.Context) {
	userID := c.Param("id")
	var req model.AdminAdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", err.Error()))
		return
	}
	tx, err := h.store.AdminAdjustUserBalance(c.Request.Context(), userID, req)
	if err != nil {
		if err == store.ErrAdminNotSupported {
			c.JSON(http.StatusNotImplemented, model.NewAPIError("not_supported", err.Error()))
			return
		}
		if err == store.ErrUserNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
			return
		}
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Balance updated", "transaction": tx})
}

// SetUserStatus PATCH /admin/users/:id/status
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	userID := c.Param("id")
	var req model.AdminSetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", err.Error()))
		return
	}
	allowed := map[string]bool{
		"active": true, "disabled": true, "banned": true, "pending_verification": true,
	}
	if !allowed[req.Status] {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid status"))
		return
	}
	if err := h.store.AdminSetUserStatus(c.Request.Context(), userID, req.Status, req.Note); err != nil {
		if err == store.ErrUserNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "status": req.Status})
}

// ReloadUserFromDB POST /admin/users/:id/reload-from-db
func (h *AdminHandler) ReloadUserFromDB(c *gin.Context) {
	userID := c.Param("id")
	if err := h.store.AdminReloadUserFromDB(c.Request.Context(), userID); err != nil {
		if err == store.ErrUserNotFound {
			c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "User not found"))
			return
		}
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", err.Error()))
		return
	}
	user, _ := h.store.AdminGetUser(c.Request.Context(), userID)
	c.JSON(http.StatusOK, gin.H{"message": "Reloaded from database into Redis", "user": user})
}

func adminPagination(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return
}
