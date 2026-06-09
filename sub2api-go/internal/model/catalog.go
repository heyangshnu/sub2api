package model

import (
	"strings"
)

type ModelKind string

const (
	ModelKindChat  ModelKind = "chat"
	ModelKindImage ModelKind = "image"
)

// PlatformModel is a user-facing model option (GPT / Gemini / Claude / Image).
type PlatformModel struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	ProviderModel string    `json:"provider_model"`
	Kind          ModelKind `json:"kind"`
	Provider      string    `json:"provider"`
}

var platformCatalog = []PlatformModel{
	{ID: "deepseek", Label: "DeepSeek", ProviderModel: "deepseek-chat", Kind: ModelKindChat, Provider: "deepseek"},
	{ID: "gpt", Label: "GPT", ProviderModel: "gpt-4o-mini", Kind: ModelKindChat, Provider: "openai"},
	{ID: "gemini", Label: "Gemini", ProviderModel: "gemini-1.5-flash", Kind: ModelKindChat, Provider: "google"},
	{ID: "claude", Label: "Claude", ProviderModel: "claude-3-5-haiku-20241022", Kind: ModelKindChat, Provider: "anthropic"},
	{ID: "image", Label: "Image", ProviderModel: "dall-e-3", Kind: ModelKindImage, Provider: "openai"},
}

// legacyUpstreamToPlatform maps old upstream ids to platform ids (CHAT_ENABLED_MODELS / subscription).
var legacyUpstreamToPlatform = map[string]string{
	"deepseek-chat":  "deepseek",
	"deepseek-coder": "deepseek",
}

// DefaultPlatformModelIDs is the default chat/key catalog when CHAT_ENABLED_MODELS is unset.
var DefaultPlatformModelIDs = []string{"deepseek", "gpt", "gemini", "claude", "image"}

func PlatformCatalog() []PlatformModel {
	out := make([]PlatformModel, len(platformCatalog))
	copy(out, platformCatalog)
	return out
}

func LookupPlatformModel(id string) (PlatformModel, bool) {
	id = strings.TrimSpace(id)
	for _, m := range platformCatalog {
		if m.ID == id {
			return m, true
		}
	}
	return PlatformModel{}, false
}

// ResolvePlatformModel maps a platform id or upstream model id to catalog entry + upstream id.
func ResolvePlatformModel(modelName string) (PlatformModel, string, bool) {
	modelName = strings.TrimSpace(modelName)
	if pm, ok := LookupPlatformModel(modelName); ok {
		return pm, pm.ProviderModel, true
	}
	for _, pm := range platformCatalog {
		if pm.ProviderModel == modelName {
			return pm, pm.ProviderModel, true
		}
	}
	if platformID, ok := legacyUpstreamToPlatform[modelName]; ok {
		if pm, ok := LookupPlatformModel(platformID); ok {
			upstream := pm.ProviderModel
			if modelName == "deepseek-coder" {
				upstream = "deepseek-coder"
			}
			return pm, upstream, true
		}
	}
	return PlatformModel{}, modelName, false
}

// CanonicalPlatformID normalizes platform or legacy upstream id to platform id.
func CanonicalPlatformID(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if pm, ok := LookupPlatformModel(modelName); ok {
		return pm.ID
	}
	if platformID, ok := legacyUpstreamToPlatform[modelName]; ok {
		return platformID
	}
	if pm, _, ok := ResolvePlatformModel(modelName); ok {
		return pm.ID
	}
	return modelName
}

func ModelKindOf(modelName string) ModelKind {
	if pm, _, ok := ResolvePlatformModel(modelName); ok {
		return pm.Kind
	}
	return ModelKindChat
}

// PricingKey returns the key used in DefaultPricing (platform id preferred).
func PricingKey(modelName string) string {
	if pm, ok := LookupPlatformModel(modelName); ok {
		return pm.ID
	}
	return modelName
}

// KeyAllowsModel returns true when the key has no allowlist (legacy) or includes the model.
func KeyAllowsModel(key *APIKey, modelName string) bool {
	if key == nil || len(key.AllowedModels) == 0 {
		return true
	}
	modelName = strings.TrimSpace(modelName)
	for _, allowed := range key.AllowedModels {
		if allowed == modelName {
			return true
		}
	}
	if pm, _, ok := ResolvePlatformModel(modelName); ok {
		for _, allowed := range key.AllowedModels {
			if allowed == pm.ID {
				return true
			}
		}
	}
	return false
}

// ModelInAllowlist checks subscription or config allowlists (supports platform ids and upstream ids).
func ModelInAllowlist(modelName string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	canonical := strings.TrimSpace(modelName)
	if pm, _, ok := ResolvePlatformModel(modelName); ok {
		canonical = pm.ID
	}
	for _, m := range allowed {
		m = strings.TrimSpace(m)
		if m == modelName || m == canonical {
			return true
		}
		if pm2, _, ok := ResolvePlatformModel(m); ok && pm2.ID == canonical {
			return true
		}
	}
	return false
}

func LastUserMessageContent(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}
