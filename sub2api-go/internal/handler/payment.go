package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

type PaymentHandler struct {
	stripeService *service.StripeService
	store         store.Store
}

func NewPaymentHandler(ss *service.StripeService, s store.Store) *PaymentHandler {
	return &PaymentHandler{
		stripeService: ss,
		store:         s,
	}
}

// CreateCheckout handles POST /v1/payment/checkout
// Creates a Stripe Checkout session for the authenticated user
func (h *PaymentHandler) CreateCheckout(c *gin.Context) {
	if h.stripeService == nil || !h.stripeService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, model.NewAPIError("service_unavailable", "Payment service not configured"))
		return
	}

	keyHash := middleware.GetKeyHash(c)

	var req struct {
		Amount float64 `json:"amount" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Amount must be between $1 and $1000"))
		return
	}

	// Get key info
	key, err := h.store.GetKeyByHash(c.Request.Context(), keyHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to get key info"))
		return
	}

	// Create checkout session
	session, err := h.stripeService.CreateCheckoutSession(key.ID, keyHash, req.Amount)
	if err != nil {
		log.Printf("Stripe checkout error: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError("payment_error", "Failed to create checkout session"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": session.URL,
		"session_id":   session.ID,
	})
}

// HandleWebhook handles POST /webhook/stripe
// Processes Stripe webhook events (payment success, etc.)
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	if h.stripeService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment service not configured"})
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	event, err := h.stripeService.VerifyWebhookSignature(payload, signature)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("Failed to parse checkout session: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
			return
		}

		// Only process paid sessions
		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			log.Printf("Session %s not paid yet, status: %s", session.ID, session.PaymentStatus)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		// Extract metadata
		keyHash := session.Metadata["key_hash"]
		amountStr := session.Metadata["amount"]
		
		if keyHash == "" || amountStr == "" {
			log.Printf("Missing metadata in session %s", session.ID)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			log.Printf("Invalid amount in metadata: %s", amountStr)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		// Topup the balance
		note := "Stripe payment: " + session.ID
		if err := h.store.Topup(c.Request.Context(), keyHash, amount, note); err != nil {
			log.Printf("Failed to topup balance for session %s: %v", session.ID, err)
			// Don't return error - Stripe will retry
			c.JSON(http.StatusOK, gin.H{"received": true, "error": "topup_failed"})
			return
		}

		log.Printf("Successfully topped up $%.2f for key %s (session: %s)", amount, keyHash[:16], session.ID)
		c.JSON(http.StatusOK, gin.H{"received": true, "topup": amount})

	default:
		// Unhandled event type
		log.Printf("Unhandled event type: %s", event.Type)
		c.JSON(http.StatusOK, gin.H{"received": true})
	}
}

// GetPaymentStatus handles GET /v1/payment/status/:session_id
// Checks the status of a payment session
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	if h.stripeService == nil || !h.stripeService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, model.NewAPIError("service_unavailable", "Payment service not configured"))
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Session ID required"))
		return
	}

	session, err := h.stripeService.GetCheckoutSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewAPIError("not_found", "Session not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":     session.ID,
		"payment_status": session.PaymentStatus,
		"amount":         float64(session.AmountTotal) / 100,
	})
}
