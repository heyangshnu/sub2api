package handler

import (
	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
)

// ModelsFromConfig builds OpenAI-compatible model list from platform catalog + providers.
func ModelsFromConfig(cfg *config.Config) []gin.H {
	if cfg == nil {
		return defaultModelList()
	}
	catalog := cfg.AvailablePlatformModels()
	if len(catalog) == 0 {
		return defaultModelList()
	}
	out := make([]gin.H, 0, len(catalog))
	for _, pm := range catalog {
		out = append(out, gin.H{
			"id":       pm.ID,
			"object":   "model",
			"owned_by": pm.Provider,
			"kind":     pm.Kind,
			"label":    pm.Label,
		})
	}
	return out
}

func defaultModelList() []gin.H {
	out := make([]gin.H, 0, len(model.PlatformCatalog()))
	for _, pm := range model.PlatformCatalog() {
		out = append(out, gin.H{
			"id":       pm.ID,
			"object":   "model",
			"owned_by": pm.Provider,
			"kind":     pm.Kind,
			"label":    pm.Label,
		})
	}
	return out
}
