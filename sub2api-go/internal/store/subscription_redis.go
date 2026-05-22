package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"sub2api-go/internal/model"
)

const KeyPrefixUserSubscription = "subscription:"

func (s *RedisStore) GetUserSubscription(ctx context.Context, userID string) (*model.UserSubscription, error) {
	raw, err := s.client.Get(ctx, KeyPrefixUserSubscription+userID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sub model.UserSubscription
	if err := json.Unmarshal([]byte(raw), &sub); err != nil {
		return nil, err
	}
	if time.Now().UTC().After(sub.PeriodEnd) {
		return nil, nil
	}
	return &sub, nil
}

func (s *RedisStore) ActivateUserSubscription(ctx context.Context, userID, planID string, periodDays int, resetSpend bool) error {
	now := time.Now().UTC()
	sub := model.UserSubscription{
		PlanID:      planID,
		PeriodStart: now,
		PeriodEnd:   now.Add(time.Duration(periodDays) * 24 * time.Hour),
		SpentUSD:    0,
	}
	if !resetSpend {
		if old, _ := s.GetUserSubscription(ctx, userID); old != nil && old.PlanID == planID {
			sub.SpentUSD = old.SpentUSD
		}
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, KeyPrefixUserSubscription+userID, b, 0).Err()
}

func (s *RedisStore) EnsureDefaultSubscription(ctx context.Context, userID, defaultPlanID string, periodDays int) error {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub != nil {
		return nil
	}
	if defaultPlanID == "" {
		return nil
	}
	return s.ActivateUserSubscription(ctx, userID, defaultPlanID, periodDays, true)
}

func (s *RedisStore) CheckSubscriptionModel(ctx context.Context, userID, modelName string, allowed []string) error {
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

func (s *RedisStore) CheckSubscriptionSpendCap(ctx context.Context, userID string, capUSD, additionalUSD float64) error {
	if capUSD <= 0 && additionalUSD > 0 {
		return ErrSubscriptionCapExceeded
	}
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

func (s *RedisStore) AddSubscriptionSpend(ctx context.Context, userID string, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	key := KeyPrefixUserSubscription + userID
	raw, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	var sub model.UserSubscription
	if json.Unmarshal([]byte(raw), &sub) != nil {
		return nil
	}
	if time.Now().UTC().After(sub.PeriodEnd) {
		return nil
	}
	sub.SpentUSD += amountUSD
	b, _ := json.Marshal(sub)
	return s.client.Set(ctx, key, b, 0).Err()
}
