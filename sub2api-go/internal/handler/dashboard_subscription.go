package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

type DashboardSubscriptionHandler struct {
	store         store.Store
	stripeService *service.StripeService
	subService    *service.SubscriptionService
	cfg           *config.Config
}

func NewDashboardSubscriptionHandler(s store.Store, ss *service.StripeService, cfg *config.Config) *DashboardSubscriptionHandler {
	return &DashboardSubscriptionHandler{
		store:         s,
		stripeService: ss,
		subService:    service.NewSubscriptionService(s, cfg),
		cfg:           cfg,
	}
}

// ListPlans GET /dashboard/subscription/plans
func (h *DashboardSubscriptionHandler) ListPlans(c *gin.Context) {
	if h.cfg == nil || !h.cfg.SubscriptionsEnabled {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "plans": []config.SubscriptionPlan{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":      true,
		"period_days":  h.cfg.SubscriptionPeriodDays,
		"plans":        h.cfg.SubscriptionPlans,
		"currency":     "USD",
	})
}

// GetSubscription GET /dashboard/subscription
func (h *DashboardSubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	view := h.subService.BuildView(c.Request.Context(), uid)
	c.JSON(http.StatusOK, gin.H{"subscription": view})
}

// CreateSubscriptionCheckout POST /dashboard/subscription/checkout
func (h *DashboardSubscriptionHandler) CreateSubscriptionCheckout(c *gin.Context) {
	if h.cfg == nil || !h.cfg.SubscriptionsEnabled {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Subscriptions are not enabled"))
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	var req model.SubscriptionCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "plan_id is required"))
		return
	}
	plan := h.cfg.PlanByID(req.PlanID)
	if plan == nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Unknown subscription plan"))
		return
	}
	if plan.MonthlyPriceUSD <= 0 {
		// Free tier: activate immediately
		_ = h.store.ActivateUserSubscription(c.Request.Context(), uid, plan.ID, h.cfg.SubscriptionPeriodDays, true)
		if plan.IncludedBalanceUSD > 0 {
			_ = h.store.AccountTopup(c.Request.Context(), uid, plan.IncludedBalanceUSD, "subscription_grant",
				"Included balance: "+plan.ID, "", true)
		}
		c.JSON(http.StatusOK, gin.H{
			"activated": true,
			"plan_id":   plan.ID,
			"message":   "Free plan activated",
		})
		return
	}

	if h.stripeService == nil || !h.stripeService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, model.NewAPIError("service_unavailable", "Payment service not configured"))
		return
	}
	session, err := h.stripeService.CreateSubscriptionCheckoutSession(uid, plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("payment_error", "Failed to create checkout session"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": session.URL, "session_id": session.ID, "plan_id": plan.ID})
}
