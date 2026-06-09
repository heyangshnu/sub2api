package config

import (
	"strings"

	"sub2api-go/internal/model"
)

func providerHasModel(cfg *Config, providerName, upstreamModel string) bool {
	if cfg == nil {
		return false
	}
	for _, p := range cfg.Providers {
		if p.Name != providerName {
			continue
		}
		for _, m := range p.Models {
			if m == upstreamModel {
				return true
			}
		}
	}
	return false
}

// AvailablePlatformModels returns catalog entries whose upstream provider is configured.
func (c *Config) AvailablePlatformModels() []model.PlatformModel {
	var out []model.PlatformModel
	for _, pm := range model.PlatformCatalog() {
		if providerHasModel(c, pm.Provider, pm.ProviderModel) {
			out = append(out, pm)
		}
	}
	return out
}

func (c *Config) AvailablePlatformModelIDs() []string {
	models := c.AvailablePlatformModels()
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ID)
	}
	return out
}

// ValidateSinglePlatformModel returns exactly one valid platform model id.
func (c *Config) ValidateSinglePlatformModel(ids []string) (string, bool) {
	validated := c.ValidatePlatformModelIDs(ids)
	if len(validated) != 1 {
		return "", false
	}
	return validated[0], true
}

func (c *Config) ValidatePlatformModelIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	available := make(map[string]bool)
	for _, pm := range c.AvailablePlatformModels() {
		available[pm.ID] = true
	}
	var out []string
	seen := make(map[string]bool)
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		if !available[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
