package handler

import (
	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
)

// ModelsFromConfig builds OpenAI-compatible model list from configured providers.
func ModelsFromConfig(cfg *config.Config) []gin.H {
	if cfg == nil {
		return defaultModelList()
	}
	seen := make(map[string]bool)
	var out []gin.H
	for _, p := range cfg.Providers {
		ownedBy := p.Name
		if ownedBy == "" {
			ownedBy = "system"
		}
		for _, m := range p.Models {
			if m == "" || seen[m] {
				continue
			}
			seen[m] = true
			out = append(out, gin.H{
				"id":       m,
				"object":   "model",
				"owned_by": ownedBy,
			})
		}
	}
	if len(out) == 0 {
		return defaultModelList()
	}
	return out
}

func defaultModelList() []gin.H {
	return []gin.H{
		{"id": "claude-3-5-sonnet-20241022", "object": "model", "owned_by": "anthropic"},
		{"id": "claude-3-5-haiku-20241022", "object": "model", "owned_by": "anthropic"},
		{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
		{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
		{"id": "deepseek-chat", "object": "model", "owned_by": "deepseek"},
	}
}
