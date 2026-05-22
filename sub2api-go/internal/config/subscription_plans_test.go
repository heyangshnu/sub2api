package config

import "testing"

func TestParseSubscriptionPlans(t *testing.T) {
	raw := "free:0:0.5:0:deepseek-chat|basic:9.99:30:5:deepseek-chat,gpt-4o-mini"
	plans, err := ParseSubscriptionPlans(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 {
		t.Fatalf("want 2 plans got %d", len(plans))
	}
	if plans[0].ID != "free" || plans[0].MonthlySpendCapUSD != 0.5 {
		t.Fatalf("free plan: %+v", plans[0])
	}
	if plans[1].MonthlyPriceUSD != 9.99 || len(plans[1].AllowedModels) != 2 {
		t.Fatalf("basic plan: %+v", plans[1])
	}
}
