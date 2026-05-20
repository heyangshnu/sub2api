package service

import (
	"context"
	"errors"

	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

// BillingService handles all billing operations
type BillingService struct {
	store store.Store
}

func NewBillingService(s store.Store) *BillingService {
	return &BillingService{store: s}
}

// EstimateCost estimates the cost for a request based on input tokens
func (s *BillingService) EstimateCost(modelName string, estimatedInputTokens int) float64 {
	pricing, ok := model.DefaultPricing[modelName]
	if !ok {
		// Default pricing if model not found
		pricing = model.ModelPricing{
			InputPricePerK:  0.003,
			OutputPricePerK: 0.015,
		}
	}
	
	// Estimate: assume output will be similar to input
	estimatedOutputTokens := estimatedInputTokens
	if estimatedOutputTokens < 100 {
		estimatedOutputTokens = 100 // Minimum estimate
	}
	
	inputCost := float64(estimatedInputTokens) / 1000.0 * pricing.InputPricePerK
	outputCost := float64(estimatedOutputTokens) / 1000.0 * pricing.OutputPricePerK
	
	return inputCost + outputCost
}

// CalculateActualCost calculates the actual cost based on usage
func (s *BillingService) CalculateActualCost(modelName string, usage model.Usage) float64 {
	pricing, ok := model.DefaultPricing[modelName]
	if !ok {
		pricing = model.ModelPricing{
			InputPricePerK:  0.003,
			OutputPricePerK: 0.015,
		}
	}
	
	inputCost := float64(usage.PromptTokens) / 1000.0 * pricing.InputPricePerK
	outputCost := float64(usage.CompletionTokens) / 1000.0 * pricing.OutputPricePerK
	
	return inputCost + outputCost
}

// PreDeduct attempts to pre-deduct estimated cost
func (s *BillingService) PreDeduct(ctx context.Context, keyHash string, amount float64) error {
	return s.store.PreDeduct(ctx, keyHash, amount)
}

// FinalizeDeduct adjusts balance based on actual usage
func (s *BillingService) FinalizeDeduct(ctx context.Context, keyHash string, preDeducted, actualAmount float64, usage model.Usage, modelName, requestID string) error {
	return s.store.FinalizeDeduct(ctx, keyHash, preDeducted, actualAmount, usage, modelName, requestID)
}

// RefundPreDeduct refunds pre-deducted amount on failure
func (s *BillingService) RefundPreDeduct(ctx context.Context, keyHash string, amount float64) error {
	return s.store.RefundPreDeduct(ctx, keyHash, amount)
}

// GetBalance returns current balance
func (s *BillingService) GetBalance(ctx context.Context, keyHash string) (float64, error) {
	return s.store.GetBalance(ctx, keyHash)
}

// CountInputTokens estimates input token count from messages
// This is a rough estimate; actual count comes from the provider
func CountInputTokens(messages []model.Message) int {
	total := 0
	for _, msg := range messages {
		// Rough estimate: 4 characters per token (for English)
		// Chinese typically 1.5-2 characters per token
		total += len(msg.Content) / 3
		total += 4 // Role overhead
	}
	return total
}

// ValidateBalance checks if user has sufficient balance
func (s *BillingService) ValidateBalance(ctx context.Context, keyHash string, estimatedCost float64) error {
	balance, err := s.store.GetBalance(ctx, keyHash)
	if err != nil {
		return err
	}

	if balance < estimatedCost {
		return errors.New("insufficient balance")
	}

	return nil
}

// PreDeductForAPI checks key spend limit then pre-deducts from user account (USD).
func (s *BillingService) PreDeductForAPI(ctx context.Context, key *model.APIKey, amount float64) error {
	if key == nil {
		return errors.New("missing api key")
	}
	if err := s.store.CheckKeySpendLimit(ctx, key.ID, key.SpendLimit, amount); err != nil {
		return err
	}
	return s.store.AccountPreDeduct(ctx, key.UserID, amount)
}

func (s *BillingService) RefundForAPI(ctx context.Context, key *model.APIKey, amount float64) error {
	if key == nil {
		return nil
	}
	return s.store.AccountRefundPreDeduct(ctx, key.UserID, amount)
}

func (s *BillingService) FinalizeForAPI(ctx context.Context, key *model.APIKey, preDeducted, actual float64, usage model.Usage, modelName, requestID string) error {
	if key == nil {
		return errors.New("missing api key")
	}
	return s.store.AccountFinalizeDeduct(ctx, key.UserID, key.ID, "api_consume", modelName, requestID, preDeducted, actual, usage)
}

// PreDeductForChat applies monthly grant (if any) then pre-deducts account for dashboard chat.
func (s *BillingService) PreDeductForChat(ctx context.Context, userID string, amount, grantUSD float64) error {
	if grantUSD > 0 {
		_, _ = s.store.TryMonthlyGrant(ctx, userID, grantUSD)
	}
	return s.store.AccountPreDeduct(ctx, userID, amount)
}

func (s *BillingService) RefundForChat(ctx context.Context, userID string, amount float64) error {
	return s.store.AccountRefundPreDeduct(ctx, userID, amount)
}

func (s *BillingService) FinalizeForChat(ctx context.Context, userID string, preDeducted, actual float64, usage model.Usage, modelName, requestID string) error {
	return s.store.AccountFinalizeDeduct(ctx, userID, "", "chat_consume", modelName, requestID, preDeducted, actual, usage)
}
