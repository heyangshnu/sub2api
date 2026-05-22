package store

import (
	"context"
	"strings"
	"time"

	"sub2api-go/internal/model"
)

func (s *MemoryStore) GetUserSubscription(ctx context.Context, userID string) (*model.UserSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subscriptions[userID]
	if !ok || sub == nil || time.Now().UTC().After(sub.PeriodEnd) {
		return nil, nil
	}
	cp := *sub
	return &cp, nil
}

func (s *MemoryStore) ActivateUserSubscription(ctx context.Context, userID, planID string, periodDays int, resetSpend bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	spent := 0.0
	if !resetSpend {
		if old, ok := s.subscriptions[userID]; ok && old != nil && old.PlanID == planID {
			spent = old.SpentUSD
		}
	}
	s.subscriptions[userID] = &model.UserSubscription{
		PlanID:      planID,
		PeriodStart: now,
		PeriodEnd:   now.Add(time.Duration(periodDays) * 24 * time.Hour),
		SpentUSD:    spent,
	}
	return nil
}

func (s *MemoryStore) EnsureDefaultSubscription(ctx context.Context, userID, defaultPlanID string, periodDays int) error {
	sub, _ := s.GetUserSubscription(ctx, userID)
	if sub != nil || defaultPlanID == "" {
		return nil
	}
	return s.ActivateUserSubscription(ctx, userID, defaultPlanID, periodDays, true)
}

func (s *MemoryStore) CheckSubscriptionModel(ctx context.Context, userID, modelName string, allowed []string) error {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionRequired
	}
	modelName = strings.TrimSpace(modelName)
	for _, m := range allowed {
		if m == modelName {
			return nil
		}
	}
	return ErrSubscriptionModelNotAllowed
}

func (s *MemoryStore) CheckSubscriptionSpendCap(ctx context.Context, userID string, capUSD, additionalUSD float64) error {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionRequired
	}
	if capUSD > 0 && sub.SpentUSD+additionalUSD > capUSD+1e-9 {
		return ErrSubscriptionCapExceeded
	}
	return nil
}

func (s *MemoryStore) AddSubscriptionSpend(ctx context.Context, userID string, amountUSD float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[userID]
	if !ok || sub == nil || time.Now().UTC().After(sub.PeriodEnd) {
		return nil
	}
	sub.SpentUSD += amountUSD
	return nil
}
