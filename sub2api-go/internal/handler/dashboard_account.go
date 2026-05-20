package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

type DashboardAccountHandler struct {
	store         store.Store
	stripeService *service.StripeService
	cfg           *config.Config
}

func NewDashboardAccountHandler(s store.Store, ss *service.StripeService, cfg *config.Config) *DashboardAccountHandler {
	return &DashboardAccountHandler{store: s, stripeService: ss, cfg: cfg}
}

// CreateAccountCheckout POST /dashboard/payment/checkout
func (h *DashboardAccountHandler) CreateAccountCheckout(c *gin.Context) {
	if h.stripeService == nil || !h.stripeService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, model.NewAPIError("service_unavailable", "Payment service not configured"))
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	var req model.AccountCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Amount must be between $1 and $1000"))
		return
	}

	session, err := h.stripeService.CreateAccountCheckoutSession(uid, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("payment_error", "Failed to create checkout session"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": session.URL, "session_id": session.ID})
}

// ListAccountTransactions GET /dashboard/account/transactions
func (h *DashboardAccountHandler) ListAccountTransactions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := parseIntDefault(v, 20); err == nil {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := parseIntDefault(v, 0); err == nil {
			offset = n
		}
	}
	txs, total, err := h.store.ListAccountTransactions(c.Request.Context(), uid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to list transactions"))
		return
	}
	if txs == nil {
		txs = []*model.Transaction{}
	}
	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

func parseIntDefault(s string, def int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def, err
	}
	return n, nil
}
