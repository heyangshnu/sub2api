package service

import (
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"sub2api-go/internal/config"
)

type StripeService struct {
	secretKey      string
	webhookSecret  string
	successURL     string
	cancelURL      string
}

func NewStripeService(secretKey, webhookSecret, successURL, cancelURL string) *StripeService {
	stripe.Key = secretKey
	return &StripeService{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

// CreateAccountCheckoutSession creates a Stripe Checkout session for user account topup (JWT).
func (s *StripeService) CreateAccountCheckoutSession(userID string, amount float64) (*stripe.CheckoutSession, error) {
	amountCents := int64(amount * 100)
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Sub2API Account Topup"),
						Description: stripe.String(fmt.Sprintf("Add $%.2f to your account balance", amount)),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.cancelURL),
		Metadata: map[string]string{
			"user_id":      userID,
			"type":         "account_topup",
			"amount":       fmt.Sprintf("%.2f", amount),
		},
	}
	return session.New(params)
}

// CreateCheckoutSession creates a Stripe Checkout session for topup
// amount is in USD (e.g., 10.00 for $10)
func (s *StripeService) CreateCheckoutSession(keyID, keyHash string, amount float64) (*stripe.CheckoutSession, error) {
	// Convert to cents
	amountCents := int64(amount * 100)

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Sub2API Balance Topup"),
						Description: stripe.String(fmt.Sprintf("Add $%.2f to your API balance", amount)),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.cancelURL),
		Metadata: map[string]string{
			"key_id":   keyID,
			"key_hash": keyHash,
			"amount":   fmt.Sprintf("%.2f", amount),
		},
	}

	return session.New(params)
}

// CreateSubscriptionCheckoutSession one-time payment for a subscription period (metadata.type=subscription).
func (s *StripeService) CreateSubscriptionCheckoutSession(userID string, plan *config.SubscriptionPlan) (*stripe.CheckoutSession, error) {
	amountCents := int64(plan.MonthlyPriceUSD * 100)
	desc := fmt.Sprintf("Plan %s: $%.2f/mo cap, models: %s",
		plan.ID, plan.MonthlySpendCapUSD, joinModels(plan.AllowedModels))
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Sub2API Subscription — " + plan.ID),
						Description: stripe.String(desc),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}&subscription=1"),
		CancelURL:  stripe.String(s.cancelURL),
		Metadata: map[string]string{
			"user_id": userID,
			"type":    "subscription",
			"plan_id": plan.ID,
			"amount":  fmt.Sprintf("%.2f", plan.MonthlyPriceUSD),
		},
	}
	return session.New(params)
}

func joinModels(models []string) string {
	if len(models) == 0 {
		return ""
	}
	out := models[0]
	for i := 1; i < len(models); i++ {
		out += ", " + models[i]
	}
	if len(out) > 120 {
		return out[:117] + "..."
	}
	return out
}

// GetCheckoutSession retrieves a checkout session by ID
func (s *StripeService) GetCheckoutSession(sessionID string) (*stripe.CheckoutSession, error) {
	return session.Get(sessionID, nil)
}

// VerifyWebhookSignature verifies the Stripe webhook signature
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, signature, s.webhookSecret)
}

// IsEnabled returns true if Stripe is configured
func (s *StripeService) IsEnabled() bool {
	return s.secretKey != "" && s.secretKey != "sk_test_xxx"
}
