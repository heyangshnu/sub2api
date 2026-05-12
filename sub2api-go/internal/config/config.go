package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	Port     string
	AdminKey string

	// Auth
	JWTSecret  string
	InviteCode string // 邀请码，空则不需要邀请码

	// Providers
	Providers []ProviderConfig

	// Redis
	RedisURL string

	// Database
	DatabaseURL string

	// Stripe
	StripeSecretKey    string
	StripeWebhookSecret string
	StripeSuccessURL   string
	StripeCancelURL    string
}

type ProviderConfig struct {
	Name     string
	APIKey   string
	BaseURL  string
	Models   []string
	Priority int
}

var cfg *Config

func Load() *Config {
	if cfg != nil {
		return cfg
	}

	cfg = &Config{
		Port:                getEnv("PORT", "3000"),
		AdminKey:            getEnv("ADMIN_KEY", "sk-admin-sub2api-secret"),
		JWTSecret:           getEnv("JWT_SECRET", "sub2api-jwt-secret-change-in-production"),
		InviteCode:          getEnv("INVITE_CODE", ""), // 空则不需要邀请码
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		DatabaseURL:         getEnv("DATABASE_URL", "sqlite://./data/sub2api.db"),
		Providers:           loadProviders(),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeSuccessURL:    getEnv("STRIPE_SUCCESS_URL", "http://localhost:3001/payment/success"),
		StripeCancelURL:     getEnv("STRIPE_CANCEL_URL", "http://localhost:3001"),
	}

	return cfg
}

func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

func loadProviders() []ProviderConfig {
	var providers []ProviderConfig

	// Anthropic
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providers = append(providers, ProviderConfig{
			Name:    "anthropic",
			APIKey:  key,
			BaseURL: getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			Models: []string{
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-opus-20240229",
			},
			Priority: 1,
		})
	}

	// OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, ProviderConfig{
			Name:    "openai",
			APIKey:  key,
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			Models: []string{
				"gpt-4o",
				"gpt-4o-mini",
				"gpt-4-turbo",
			},
			Priority: 2,
		})
	}

	// DeepSeek
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		providers = append(providers, ProviderConfig{
			Name:    "deepseek",
			APIKey:  key,
			BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			Models: []string{
				"deepseek-chat",
				"deepseek-coder",
			},
			Priority: 3,
		})
	}

	return providers
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvSlice(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}
	return strings.Split(val, ",")
}
