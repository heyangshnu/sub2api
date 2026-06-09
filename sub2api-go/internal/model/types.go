package model

import "time"

// ==================== OpenAI Compatible Types ====================

// ChatCompletionRequest represents the request body for /v1/chat/completions
type ChatCompletionRequest struct {
	Model            string         `json:"model" binding:"required"`
	Messages         []Message      `json:"messages" binding:"required"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	N                int            `json:"n,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	Stop             interface{}    `json:"stop,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	User             string         `json:"user,omitempty"`
}

type Message struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionResponse represents the response from /v1/chat/completions
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

type Choice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	Delta        *Delta   `json:"delta,omitempty"`
	LogProbs     any      `json:"logprobs,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ==================== Streaming Types ====================

type StreamChunk struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// ==================== User Types ====================

type User struct {
	ID                    string     `json:"id" db:"id"`
	Email                 string     `json:"email" db:"email"`
	PasswordHash          string     `json:"-" db:"password_hash"`
	Name                  string     `json:"name" db:"name"`
	Status                string     `json:"status" db:"status"` // pending_verification, active, disabled
	EmailVerified         bool       `json:"email_verified" db:"email_verified"`
	EmailVerifyTokenHash  string     `json:"-" db:"email_verify_token_hash"`
	EmailVerifyExpiresAt  *time.Time `json:"-" db:"email_verify_expires_at"`
	Balance               float64    `json:"balance" db:"balance"`
	RechargedBalance      float64    `json:"recharged_balance,omitempty" db:"recharged_balance"`
	HasPaid               bool       `json:"has_paid" db:"has_paid"`
	FirstPaidAt           *time.Time `json:"first_paid_at,omitempty" db:"first_paid_at"`
	LastMonthlyGrantMonth string     `json:"last_monthly_grant_month,omitempty" db:"last_monthly_grant_month"`
	TermsAcceptedAt       *time.Time `json:"terms_accepted_at,omitempty" db:"terms_accepted_at"`
	TermsVersion          string     `json:"terms_version,omitempty" db:"terms_version"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// UserProfile is returned by GET /dashboard/me (no password).
type UserProfile struct {
	ID                    string  `json:"id"`
	Email                 string  `json:"email"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	Balance          float64 `json:"balance"`           // 展示用：仅客户真实充值结余（不含月赠）
	SpendableBalance float64 `json:"spendable_balance"` // 实际可消费总额（含月赠等）
	HasPaid          bool    `json:"has_paid"`
	CanCreateKey     bool    `json:"can_create_key"`
	Currency         string  `json:"currency"`
	Subscription     *UserSubscriptionView `json:"subscription,omitempty"`
}

// UserSubscription is persisted per user (Redis).
type UserSubscription struct {
	PlanID      string    `json:"plan_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	SpentUSD    float64   `json:"spent_usd"`
}

// UserSubscriptionView is returned to the dashboard.
type UserSubscriptionView struct {
	PlanID             string    `json:"plan_id"`
	MonthlyPriceUSD    float64   `json:"monthly_price_usd"`
	MonthlySpendCapUSD float64   `json:"monthly_spend_cap_usd"`
	SpentThisPeriod    float64   `json:"spent_this_period"`
	RemainingCapUSD    float64   `json:"remaining_cap_usd"`
	AllowedModels      []string  `json:"allowed_models"`
	PeriodEnd          time.Time `json:"period_end"`
	Active             bool      `json:"active"`
}

type SubscriptionCheckoutRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=6"`
	Name             string `json:"name,omitempty"`
	VerificationCode string `json:"verification_code,omitempty"` // required when server has EMAIL_VERIFY_ENABLED=true
	TermsAccepted    bool   `json:"terms_accepted"`
	TermsVersion     string `json:"terms_version" binding:"required"`
}

type SendRegisterCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SendResetPasswordCodeRequest is POST /auth/send-reset-password-code (same shape as register code).
type SendResetPasswordCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest is POST /auth/reset-password after email OTP verification.
type ResetPasswordRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required"`
	NewPassword      string `json:"new_password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token  string `json:"token,omitempty"`
	User   *User  `json:"user,omitempty"`
	APIKey string `json:"api_key,omitempty"` // 首次注册时返回
}

// ==================== API Key Types ====================

type APIKey struct {
	ID           string    `json:"id" db:"id"`
	KeyHash      string    `json:"-" db:"key_hash"`
	KeyPrefix    string    `json:"key_prefix" db:"key_prefix"`
	UserID       string    `json:"user_id" db:"user_id"`
	Name         string    `json:"name" db:"name"`
	Balance      float64   `json:"balance" db:"balance"` // legacy; billing uses user account
	Status       string    `json:"status" db:"status"` // active, disabled
	RateLimit    int       `json:"rate_limit" db:"rate_limit"`
	SpendLimit   *float64  `json:"spend_limit,omitempty" db:"spend_limit"`
	SpentTotal   float64   `json:"spent_total" db:"spent_total"`
	IPWhitelist  []string  `json:"ip_whitelist,omitempty" db:"-"` // IP 白名单，空数组表示不限制
	AllowedModels []string `json:"allowed_models,omitempty" db:"-"` // 新建 Key 仅 1 个；空 = 兼容旧 Key（全部可用）
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// ==================== Transaction Types ====================

type Transaction struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	KeyID           string    `json:"key_id,omitempty" db:"key_id"`
	Type            string    `json:"type" db:"type"` // topup, admin_topup, monthly_grant, chat_consume, api_consume, admin_adjust, refund
	Amount          float64   `json:"amount" db:"amount"`
	BalanceBefore   float64   `json:"balance_before" db:"balance_before"`
	BalanceAfter    float64   `json:"balance_after" db:"balance_after"`
	Model           string    `json:"model,omitempty" db:"model"`
	InputTokens     int       `json:"input_tokens,omitempty" db:"input_tokens"`
	OutputTokens    int       `json:"output_tokens,omitempty" db:"output_tokens"`
	RequestID       string    `json:"request_id,omitempty" db:"request_id"`
	StripePaymentID string    `json:"stripe_payment_id,omitempty" db:"stripe_payment_id"`
	Note            string    `json:"note,omitempty" db:"note"`
	Actor           string    `json:"actor,omitempty" db:"actor"` // system, user, admin, stripe_webhook
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// AdminAdjustBalanceRequest PATCH /admin/users/:id/balance
type AdminAdjustBalanceRequest struct {
	SpendableBalance *float64 `json:"spendable_balance"`
	RechargedBalance *float64 `json:"recharged_balance"`
	AdjustAmount     *float64 `json:"adjust_amount"`
	Note             string   `json:"note"`
}

// AdminSetStatusRequest PATCH /admin/users/:id/status
type AdminSetStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

// DailyUsagePoint aggregates per-day consume amounts (UTC date) for Dashboard charts.
type DailyUsagePoint struct {
	Date          string  `json:"date"`           // YYYY-MM-DD UTC
	TotalConsumed float64 `json:"total_consumed"` // USD
	RequestCount  int     `json:"request_count"`
}

// UsageSummary is account-level usage stats (UTC day/month and all-time).
type UsageSummary struct {
	TodaySpendUSD      float64 `json:"today_spend_usd"`
	TodayRequestCount  int     `json:"today_request_count"`
	TodayInputTokens   int64   `json:"today_input_tokens"`
	TodayOutputTokens  int64   `json:"today_output_tokens"`
	MonthSpendUSD      float64 `json:"month_spend_usd"`
	MonthRequestCount  int     `json:"month_request_count"`
	TotalSpendUSD      float64 `json:"total_spend_usd"`
	TotalRequestCount  int     `json:"total_request_count"`
	TotalInputTokens   int64   `json:"total_input_tokens"`
	TotalOutputTokens  int64   `json:"total_output_tokens"`
}

// ModelUsageRow aggregates consume by model for a period.
type ModelUsageRow struct {
	Model         string  `json:"model"`
	RequestCount  int     `json:"request_count"`
	InputTokens   int64   `json:"input_tokens"`
	OutputTokens  int64   `json:"output_tokens"`
	TotalConsumed float64 `json:"total_consumed"`
}

// PaymentRecord is a completed top-up row for the dashboard.
type PaymentRecord struct {
	ID              string    `json:"id"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	StripeSessionID string    `json:"stripe_session_id,omitempty"`
	Note            string    `json:"note,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// RequestLogEntry is a lightweight chat audit row (no prompt / response body).
type RequestLogEntry struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id,omitempty"`
	KeyID         string    `json:"key_id"`
	RequestID     string    `json:"request_id"`
	Model         string    `json:"model"`
	Stream        bool      `json:"stream"`
	Outcome       string    `json:"outcome"` // success, insufficient_balance, upstream_error, client_error, stream_error, internal_error
	InputTokens   int       `json:"input_tokens,omitempty"`
	OutputTokens  int       `json:"output_tokens,omitempty"`
	ChargedUSD    float64   `json:"charged_usd,omitempty"`
	LedgerEntryID string    `json:"ledger_entry_id,omitempty"`
	LatencyMs     int64     `json:"latency_ms"`
	ClientIP      string    `json:"client_ip,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ==================== Model Pricing ====================

type ModelPricing struct {
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
	InputPricePerK  float64 `json:"input_price_per_k"`  // USD per 1K tokens
	OutputPricePerK float64 `json:"output_price_per_k"` // USD per 1K tokens
	FlatPriceUSD    float64 `json:"flat_price_usd,omitempty"` // per request (e.g. image)
}

// Default pricing table (USD per 1K tokens, or flat per request)
var DefaultPricing = map[string]ModelPricing{
	// Platform ids (preferred for billing)
	"deepseek": {Model: "deepseek", Provider: "deepseek", InputPricePerK: 0.00014, OutputPricePerK: 0.00028},
	"gpt":      {Model: "gpt", Provider: "openai", InputPricePerK: 0.00015, OutputPricePerK: 0.0006},
	"gemini": {Model: "gemini", Provider: "google", InputPricePerK: 0.000075, OutputPricePerK: 0.0003},
	"claude": {Model: "claude", Provider: "anthropic", InputPricePerK: 0.001, OutputPricePerK: 0.005},
	"image":  {Model: "image", Provider: "openai", FlatPriceUSD: 0.04},
	// Upstream ids
	"claude-3-5-sonnet-20241022": {Model: "claude-3-5-sonnet-20241022", Provider: "anthropic", InputPricePerK: 0.003, OutputPricePerK: 0.015},
	"claude-3-5-haiku-20241022":  {Model: "claude-3-5-haiku-20241022", Provider: "anthropic", InputPricePerK: 0.001, OutputPricePerK: 0.005},
	"claude-3-opus-20240229":     {Model: "claude-3-opus-20240229", Provider: "anthropic", InputPricePerK: 0.015, OutputPricePerK: 0.075},
	"gpt-4o":                     {Model: "gpt-4o", Provider: "openai", InputPricePerK: 0.005, OutputPricePerK: 0.015},
	"gpt-4o-mini":                {Model: "gpt-4o-mini", Provider: "openai", InputPricePerK: 0.00015, OutputPricePerK: 0.0006},
	"gpt-4-turbo":                {Model: "gpt-4-turbo", Provider: "openai", InputPricePerK: 0.01, OutputPricePerK: 0.03},
	"dall-e-3":                   {Model: "dall-e-3", Provider: "openai", FlatPriceUSD: 0.04},
	"gemini-1.5-flash":           {Model: "gemini-1.5-flash", Provider: "google", InputPricePerK: 0.000075, OutputPricePerK: 0.0003},
	"gemini-1.5-pro":             {Model: "gemini-1.5-pro", Provider: "google", InputPricePerK: 0.00125, OutputPricePerK: 0.005},
	"deepseek-chat":              {Model: "deepseek-chat", Provider: "deepseek", InputPricePerK: 0.00014, OutputPricePerK: 0.00028},
	"deepseek-coder":             {Model: "deepseek-coder", Provider: "deepseek", InputPricePerK: 0.00014, OutputPricePerK: 0.00028},
}

// HasDefaultPricing reports whether the model has an explicit row in DefaultPricing (strict billing catalog).
func HasDefaultPricing(modelName string) bool {
	_, ok := DefaultPricing[PricingKey(modelName)]
	return ok
}

// ==================== Admin Request Types ====================

type CreateKeyRequest struct {
	UserID    string  `json:"user_id" binding:"required"`
	Name      string  `json:"name,omitempty"`
	Balance   float64 `json:"balance" binding:"required,min=0"`
	RateLimit int     `json:"rate_limit,omitempty"`
}

type CreateKeyResponse struct {
	Key     string  `json:"key"`     // Only returned once on creation
	KeyID   string  `json:"key_id"`
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

type TopupRequest struct {
	Amount float64 `json:"amount" binding:"required,min=0.01"`
	Note   string  `json:"note,omitempty"`
}

// CreateUserKeyRequest 用户从 Dashboard 创建 Key 的请求（需二次验证密码）
type CreateUserKeyRequest struct {
	Name          string   `json:"name,omitempty"`
	Password      string   `json:"password" binding:"required"` // 二次验证
	RateLimit     int      `json:"rate_limit,omitempty"`
	SpendLimit    *float64 `json:"spend_limit,omitempty"`
	AllowedModels []string `json:"allowed_models" binding:"required,len=1"` // 必须且只能绑定 1 个模型
}

// UpdateKeySettingsRequest 更新 Key 设置（IP 白名单、频次）
type UpdateKeySettingsRequest struct {
	IPWhitelist   []string `json:"ip_whitelist"`
	RateLimit     int      `json:"rate_limit,omitempty"`
	SpendLimit    *float64 `json:"spend_limit,omitempty"` // nil = clear limit; omit = unchanged
	AllowedModels []string `json:"allowed_models,omitempty"`
}

type AccountCheckoutRequest struct {
	Amount float64 `json:"amount" binding:"required,min=1,max=1000"`
}

// ==================== Error Types ====================

type APIError struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

func NewAPIError(errType, message string) APIError {
	return APIError{
		Error: ErrorDetail{
			Type:    errType,
			Message: message,
		},
	}
}

// ==================== Usage Response ====================

type UsageResponse struct {
	Balance       float64 `json:"balance"`
	TotalUsed     float64 `json:"total_used"`
	RequestCount  int     `json:"request_count"`
	LastRequestAt string  `json:"last_request_at,omitempty"`
}
