package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"sub2api-go/internal/model"
)

type Config struct {
	// Server
	Port     string
	AdminKey string

	// Auth
	JWTSecret  string
	InviteCode string // 邀请码，空则不需要邀请码
	AppEnv     string

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

	// Email verification
	EmailVerifyEnabled bool
	EmailVerifyBaseURL string
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	SMTPFrom           string

	// Security / ops
	TrustedProxies []string // CIDRs for gin.SetTrustedProxies (from TRUSTED_PROXIES)
	// AllowMemoryStore: when false, Redis connection failure aborts startup in any env.
	AllowMemoryStore bool
	// RateLimitRedisFailOpen: when false, Redis errors in API rate limiter return 503 instead of allowing traffic.
	RateLimitRedisFailOpen bool
	// AllowUnknownModelPricing: when false, /v1/chat rejects models not listed in model.DefaultPricing.
	AllowUnknownModelPricing bool

	// Account wallet (USD)
	AccountMonthlyGrantUSD           float64
	RequirePaymentBeforeCreateKey    bool
	ChatEnabledModels              []string
	RegisterOTPCooldownSec           int

	// Subscriptions (optional; configured via SUBSCRIPTION_PLANS)
	SubscriptionsEnabled   bool
	SubscriptionPeriodDays int
	SubscriptionPlans      []SubscriptionPlan
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

	allowMem := getEnv("ALLOW_MEMORY_STORE", "true") != "false"
	rlFailOpen := getEnv("RATE_LIMIT_REDIS_FAIL_OPEN", "true") != "false"
	allowUnknownModel := getEnv("ALLOW_UNKNOWN_MODEL_PRICING", "true") != "false"

	subEnabled, subPeriod, subPlans, subErr := loadSubscriptionConfig()
	if subErr != nil {
		panic(fmt.Sprintf("invalid SUBSCRIPTION_PLANS: %v", subErr))
	}

	cfg = &Config{
		Port:                getEnv("PORT", "3000"),
		AppEnv:              getEnv("APP_ENV", "development"),
		AdminKey:            getEnv("ADMIN_KEY", "sk-admin-sub2api-secret"),
		JWTSecret:           getEnv("JWT_SECRET", "sub2api-jwt-secret-change-in-production"),
		InviteCode:          strings.TrimSpace(getEnv("INVITE_CODE", "")), // 空则不需要邀请码
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		DatabaseURL:         getEnv("DATABASE_URL", "sqlite://./data/sub2api.db"),
		Providers:           loadProviders(),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeSuccessURL:    getEnv("STRIPE_SUCCESS_URL", "http://localhost:3001/payment/success"),
		StripeCancelURL:     getEnv("STRIPE_CANCEL_URL", "http://localhost:3001"),
		EmailVerifyEnabled:  getEnv("EMAIL_VERIFY_ENABLED", "false") == "true",
		EmailVerifyBaseURL:  getEnv("EMAIL_VERIFY_BASE_URL", "http://localhost:3001"),
		SMTPHost:            getEnv("SMTP_HOST", ""),
		SMTPPort:            getEnvInt("SMTP_PORT", 587),
		SMTPUsername:        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:        getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:            getEnv("SMTP_FROM", ""),
		TrustedProxies:      getEnvSliceTrimmed("TRUSTED_PROXIES"),
		AllowMemoryStore:    allowMem,
		RateLimitRedisFailOpen: rlFailOpen,
		AllowUnknownModelPricing: allowUnknownModel,
		AccountMonthlyGrantUSD:    getEnvFloat("ACCOUNT_MONTHLY_GRANT_USD", 0.1),
		RequirePaymentBeforeCreateKey: getEnv("REQUIRE_PAYMENT_BEFORE_CREATE_KEY", "true") != "false",
		ChatEnabledModels:         getEnvSliceTrimmed("CHAT_ENABLED_MODELS"),
		RegisterOTPCooldownSec:    getEnvInt("REGISTER_OTP_COOLDOWN_SEC", 60),
		SubscriptionsEnabled:      subEnabled,
		SubscriptionPeriodDays:    subPeriod,
		SubscriptionPlans:         subPlans,
	}
	if len(cfg.ChatEnabledModels) == 0 {
		cfg.ChatEnabledModels = append([]string(nil), model.DefaultPlatformModelIDs...)
	}

	return cfg
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func (c *Config) ValidateProductionSecrets() error {
	if !c.IsProduction() {
		return nil
	}
	if c.AdminKey == "" || c.AdminKey == "sk-admin-sub2api-secret" {
		return errors.New("ADMIN_KEY is required in production and cannot use default value")
	}
	if c.JWTSecret == "" || c.JWTSecret == "sub2api-jwt-secret-change-in-production" {
		return errors.New("JWT_SECRET is required in production and cannot use default value")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 chars in production")
	}
	return nil
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

	// OpenAI (GPT + Image)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, ProviderConfig{
			Name:    "openai",
			APIKey:  key,
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			Models: []string{
				"gpt-4o",
				"gpt-4o-mini",
				"gpt-4-turbo",
				"dall-e-3",
			},
			Priority: 2,
		})
	}

	// Google Gemini (OpenAI-compatible endpoint)
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		providers = append(providers, ProviderConfig{
			Name:    "google",
			APIKey:  key,
			BaseURL: getEnv("GOOGLE_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai"),
			Models: []string{
				"gemini-1.5-flash",
				"gemini-1.5-pro",
				"gemini-2.0-flash",
			},
			Priority: 4,
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

func getEnvSliceTrimmed(key string) []string {
	parts := getEnvSlice(key)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
