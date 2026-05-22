package service

import (
	"context"
	"errors"

	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

// SubscriptionService enforces plan limits when SUBSCRIPTIONS_ENABLED=true.
type SubscriptionService struct {
	store store.Store
	cfg   *config.Config
}

func NewSubscriptionService(s store.Store, cfg *config.Config) *SubscriptionService {
	return &SubscriptionService{store: s, cfg: cfg}
}

func (s *SubscriptionService) Enabled() bool {
	return s.cfg != nil && s.cfg.SubscriptionsEnabled && len(s.cfg.SubscriptionPlans) > 0
}

func (s *SubscriptionService) EnsureUser(ctx context.Context, userID string) error {
	if !s.Enabled() {
		return nil
	}
	free := s.cfg.FreePlan()
	if free != nil {
		return s.store.EnsureDefaultSubscription(ctx, userID, free.ID, s.cfg.SubscriptionPeriodDays)
	}
	return nil
}

func (s *SubscriptionService) EnforceBeforeRequest(ctx context.Context, userID, modelName string, estimatedCost float64) error {
	if !s.Enabled() {
		return nil
	}
	if err := s.EnsureUser(ctx, userID); err != nil {
		return err
	}
	sub, err := s.store.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return store.ErrSubscriptionRequired
	}
	plan := s.cfg.PlanByID(sub.PlanID)
	if plan == nil {
		return store.ErrSubscriptionPlanNotFound
	}
	if err := s.store.CheckSubscriptionModel(ctx, userID, modelName, plan.AllowedModels); err != nil {
		return err
	}
	return s.store.CheckSubscriptionSpendCap(ctx, userID, plan.MonthlySpendCapUSD, estimatedCost)
}

func (s *SubscriptionService) RecordSpend(ctx context.Context, userID string, amount float64) error {
	if !s.Enabled() || amount <= 0 {
		return nil
	}
	return s.store.AddSubscriptionSpend(ctx, userID, amount)
}

func (s *SubscriptionService) BuildView(ctx context.Context, userID string) *model.UserSubscriptionView {
	if !s.Enabled() {
		return nil
	}
	_ = s.EnsureUser(ctx, userID)
	sub, err := s.store.GetUserSubscription(ctx, userID)
	if err != nil || sub == nil {
		return &model.UserSubscriptionView{Active: false}
	}
	plan := s.cfg.PlanByID(sub.PlanID)
	if plan == nil {
		return &model.UserSubscriptionView{PlanID: sub.PlanID, Active: false}
	}
	remaining := plan.MonthlySpendCapUSD - sub.SpentUSD
	if remaining < 0 {
		remaining = 0
	}
	return &model.UserSubscriptionView{
		PlanID:             plan.ID,
		MonthlyPriceUSD:    plan.MonthlyPriceUSD,
		MonthlySpendCapUSD: plan.MonthlySpendCapUSD,
		SpentThisPeriod:    sub.SpentUSD,
		RemainingCapUSD:    remaining,
		AllowedModels:      plan.AllowedModels,
		PeriodEnd:          sub.PeriodEnd,
		Active:             true,
	}
}

func SubscriptionAPIError(err error) (status int, apiType, message string) {
	switch {
	case errors.Is(err, store.ErrSubscriptionRequired):
		return 402, "subscription_required", "Active subscription required. Subscribe to a plan first."
	case errors.Is(err, store.ErrSubscriptionCapExceeded):
		return 402, "subscription_cap_exceeded", "Monthly subscription spend cap reached. Upgrade plan or wait for next period."
	case errors.Is(err, store.ErrSubscriptionModelNotAllowed):
		return 403, "model_not_allowed", "This model is not included in your subscription plan."
	case errors.Is(err, store.ErrSubscriptionPlanNotFound):
		return 400, "invalid_plan", "Subscription plan not found."
	default:
		return 500, "internal_error", err.Error()
	}
}
