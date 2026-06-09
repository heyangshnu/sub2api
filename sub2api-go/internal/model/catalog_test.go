package model

import "testing"

func TestResolvePlatformModel(t *testing.T) {
	pm, upstream, ok := ResolvePlatformModel("deepseek")
	if !ok || pm.ID != "deepseek" || upstream != "deepseek-chat" {
		t.Fatalf("deepseek resolve: %+v %s %v", pm, upstream, ok)
	}
	_, upstreamLegacy, okLegacy := ResolvePlatformModel("deepseek-chat")
	if !okLegacy || upstreamLegacy != "deepseek-chat" {
		t.Fatalf("deepseek-chat legacy: %s %v", upstreamLegacy, okLegacy)
	}
	pmGpt, upstreamGpt, okGpt := ResolvePlatformModel("gpt")
	if !okGpt || pmGpt.ID != "gpt" || upstreamGpt != "gpt-4o-mini" {
		t.Fatalf("gpt resolve: %+v %s %v", pmGpt, upstreamGpt, okGpt)
	}
	_, upstream2, ok2 := ResolvePlatformModel("gpt-4o-mini")
	if !ok2 || upstream2 != "gpt-4o-mini" {
		t.Fatalf("upstream resolve: %s %v", upstream2, ok2)
	}
}

func TestModelInAllowlist(t *testing.T) {
	allowed := []string{"gpt", "claude"}
	if !ModelInAllowlist("gpt-4o-mini", allowed) {
		t.Fatal("expected gpt-4o-mini allowed via gpt")
	}
	if ModelInAllowlist("gemini", allowed) {
		t.Fatal("gemini should not be allowed")
	}
}

func TestKeyAllowsModel(t *testing.T) {
	key := &APIKey{AllowedModels: []string{"claude"}}
	if !KeyAllowsModel(key, "claude") {
		t.Fatal("claude should be allowed")
	}
	if KeyAllowsModel(key, "gpt") {
		t.Fatal("gpt should be denied")
	}
}
