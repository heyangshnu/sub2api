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
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

type RegisterRequest struct {
	Email              string `json:"email" binding:"required,email"`
	Password           string `json:"password" binding:"required,min=6"`
	Name               string `json:"name,omitempty"`
	InviteCode         string `json:"invite_code,omitempty"`
	VerificationCode   string `json:"verification_code,omitempty"` // required when server has EMAIL_VERIFY_ENABLED=true
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
	Balance      float64   `json:"balance" db:"balance"`
	Status       string    `json:"status" db:"status"` // active, disabled
	RateLimit    int       `json:"rate_limit" db:"rate_limit"`
	IPWhitelist  []string  `json:"ip_whitelist,omitempty" db:"-"` // IP 白名单，空数组表示不限制
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// ==================== Transaction Types ====================

type Transaction struct {
	ID              string    `json:"id" db:"id"`
	KeyID           string    `json:"key_id" db:"key_id"`
	Type            string    `json:"type" db:"type"` // consume, topup, refund
	Amount          float64   `json:"amount" db:"amount"`
	BalanceBefore   float64   `json:"balance_before" db:"balance_before"`
	BalanceAfter    float64   `json:"balance_after" db:"balance_after"`
	Model           string    `json:"model,omitempty" db:"model"`
	InputTokens     int       `json:"input_tokens,omitempty" db:"input_tokens"`
	OutputTokens    int       `json:"output_tokens,omitempty" db:"output_tokens"`
	RequestID       string    `json:"request_id,omitempty" db:"request_id"`
	StripePaymentID string    `json:"stripe_payment_id,omitempty" db:"stripe_payment_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// DailyUsagePoint aggregates per-day consume amounts (UTC date) for Dashboard charts.
type DailyUsagePoint struct {
	Date          string  `json:"date"`           // YYYY-MM-DD UTC
	TotalConsumed float64 `json:"total_consumed"` // USD
	RequestCount  int     `json:"request_count"`
}

// RequestLogEntry is a lightweight chat audit row (no prompt / response body).
type RequestLogEntry struct {
	ID        string    `json:"id"`
	KeyID     string    `json:"key_id"`
	RequestID string    `json:"request_id"`
	Model     string    `json:"model"`
	Stream    bool      `json:"stream"`
	Outcome   string    `json:"outcome"` // success, insufficient_balance, upstream_error, client_error, stream_error, internal_error
	LatencyMs int64     `json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
}

// ==================== Model Pricing ====================

type ModelPricing struct {
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
	InputPricePerK  float64 `json:"input_price_per_k"`  // USD per 1K tokens
	OutputPricePerK float64 `json:"output_price_per_k"` // USD per 1K tokens
}

// Default pricing table (USD per 1K tokens)
var DefaultPricing = map[string]ModelPricing{
	"claude-3-5-sonnet-20241022": {Model: "claude-3-5-sonnet-20241022", Provider: "anthropic", InputPricePerK: 0.003, OutputPricePerK: 0.015},
	"claude-3-5-haiku-20241022":  {Model: "claude-3-5-haiku-20241022", Provider: "anthropic", InputPricePerK: 0.001, OutputPricePerK: 0.005},
	"claude-3-opus-20240229":     {Model: "claude-3-opus-20240229", Provider: "anthropic", InputPricePerK: 0.015, OutputPricePerK: 0.075},
	"gpt-4o":                     {Model: "gpt-4o", Provider: "openai", InputPricePerK: 0.005, OutputPricePerK: 0.015},
	"gpt-4o-mini":                {Model: "gpt-4o-mini", Provider: "openai", InputPricePerK: 0.00015, OutputPricePerK: 0.0006},
	"gpt-4-turbo":                {Model: "gpt-4-turbo", Provider: "openai", InputPricePerK: 0.01, OutputPricePerK: 0.03},
	"deepseek-chat":              {Model: "deepseek-chat", Provider: "deepseek", InputPricePerK: 0.00014, OutputPricePerK: 0.00028},
	"deepseek-coder":             {Model: "deepseek-coder", Provider: "deepseek", InputPricePerK: 0.00014, OutputPricePerK: 0.00028},
}

// HasDefaultPricing reports whether the model has an explicit row in DefaultPricing (strict billing catalog).
func HasDefaultPricing(modelName string) bool {
	_, ok := DefaultPricing[modelName]
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
	Name      string `json:"name,omitempty"`
	Password  string `json:"password" binding:"required"` // 二次验证
	RateLimit int    `json:"rate_limit,omitempty"`
}

// UpdateKeySettingsRequest 更新 Key 设置（IP 白名单、频次）
type UpdateKeySettingsRequest struct {
	IPWhitelist []string `json:"ip_whitelist"`
	RateLimit   int      `json:"rate_limit,omitempty"`
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
