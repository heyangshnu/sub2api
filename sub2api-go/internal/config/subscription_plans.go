package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SubscriptionPlan is one tier from SUBSCRIPTION_PLANS env.
type SubscriptionPlan struct {
	ID                 string   `json:"id"`
	MonthlyPriceUSD    float64  `json:"monthly_price_usd"`
	MonthlySpendCapUSD float64  `json:"monthly_spend_cap_usd"`
	IncludedBalanceUSD float64  `json:"included_balance_usd"` // credited on subscribe (one-time per checkout)
	AllowedModels      []string `json:"allowed_models"`
}

// ParseSubscriptionPlans reads SUBSCRIPTION_PLANS.
//
// Format (pipe-separated plans, comma-separated models):
//
//	id:price_usd:monthly_spend_cap:included_balance:model1,model2|...
//
// Example:
//
//	free:0:0.5:0:deepseek-chat|basic:9.99:30:5:deepseek-chat,gpt-4o-mini|pro:29.99:150:20:deepseek-chat,gpt-4o-mini,claude-3-5-haiku-20241022
func ParseSubscriptionPlans(raw string) ([]SubscriptionPlan, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var plans []SubscriptionPlan
	for _, segment := range strings.Split(raw, "|") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		parts := strings.Split(segment, ":")
		if len(parts) < 4 {
			return nil, fmt.Errorf("invalid subscription plan segment %q: need id:price:spend_cap:included[:models]", segment)
		}
		id := strings.TrimSpace(parts[0])
		if id == "" {
			return nil, fmt.Errorf("subscription plan id empty in %q", segment)
		}
		price, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || price < 0 {
			return nil, fmt.Errorf("invalid price in plan %q", id)
		}
		spendCap, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil || spendCap < 0 {
			return nil, fmt.Errorf("invalid spend_cap in plan %q", id)
		}
		included, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err != nil || included < 0 {
			return nil, fmt.Errorf("invalid included_balance in plan %q", id)
		}
		var models []string
		if len(parts) >= 5 {
			for _, m := range strings.Split(parts[4], ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					models = append(models, m)
				}
			}
		}
		if len(models) == 0 {
			return nil, fmt.Errorf("plan %q has no allowed models", id)
		}
		plans = append(plans, SubscriptionPlan{
			ID:                 id,
			MonthlyPriceUSD:    price,
			MonthlySpendCapUSD: spendCap,
			IncludedBalanceUSD: included,
			AllowedModels:      models,
		})
	}
	return plans, nil
}

func (c *Config) PlanByID(planID string) *SubscriptionPlan {
	for i := range c.SubscriptionPlans {
		if c.SubscriptionPlans[i].ID == planID {
			return &c.SubscriptionPlans[i]
		}
	}
	return nil
}

func (c *Config) FreePlan() *SubscriptionPlan {
	return c.PlanByID("free")
}

func loadSubscriptionConfig() (enabled bool, periodDays int, plans []SubscriptionPlan, err error) {
	enabled = os.Getenv("SUBSCRIPTIONS_ENABLED") == "true"
	periodDays = 30
	if v := os.Getenv("SUBSCRIPTION_PERIOD_DAYS"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			periodDays = n
		}
	}
	plans, err = ParseSubscriptionPlans(os.Getenv("SUBSCRIPTION_PLANS"))
	return enabled, periodDays, plans, err
}
