package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/config"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

type ChatHandler struct {
	providerService *service.ProviderService
	billingService  *service.BillingService
	subService      *service.SubscriptionService
	store           store.Store
	cfg             *config.Config
}

func NewChatHandler(ps *service.ProviderService, bs *service.BillingService, s store.Store, cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		providerService: ps,
		billingService:  bs,
		subService:      service.NewSubscriptionService(s, cfg),
		store:           s,
		cfg:             cfg,
	}
}

func (h *ChatHandler) respondSubscriptionError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	status, typ, msg := service.SubscriptionAPIError(err)
	c.JSON(status, model.NewAPIError(typ, msg))
	return true
}

func (h *ChatHandler) appendChatAudit(c *gin.Context, apiKey *model.APIKey, modelName string, stream bool, outcome string, startAt time.Time, requestID string) {
	if apiKey == nil {
		return
	}
	entry := &model.RequestLogEntry{
		KeyID:     apiKey.ID,
		RequestID: requestID,
		Model:     modelName,
		Stream:    stream,
		Outcome:   outcome,
		LatencyMs: time.Since(startAt).Milliseconds(),
	}
	if err := h.store.AppendRequestLog(c.Request.Context(), entry); err != nil {
		log.Printf("[%s] AppendRequestLog: %v", requestID, err)
	}
}

// ChatCompletions handles POST /v1/chat/completions
func (h *ChatHandler) ChatCompletions(c *gin.Context) {
	var req model.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request body: "+err.Error()))
		return
	}

	if h.cfg != nil && !h.cfg.AllowUnknownModelPricing && !model.HasDefaultPricing(req.Model) {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error",
			"Unknown model for billing: set ALLOW_UNKNOWN_MODEL_PRICING=true or add pricing for this model"))
		return
	}

	startAt := time.Now()
	var apiKey *model.APIKey
	if v, ok := c.Get(string(middleware.KeyAPIKey)); ok && v != nil {
		if ak, ok2 := v.(*model.APIKey); ok2 {
			apiKey = ak
		}
	}

	requestID, _ := c.Get("request_id")
	reqID := fmt.Sprintf("%v", requestID)

	// Estimate cost and pre-deduct
	estimatedTokens := service.CountInputTokens(req.Messages)
	estimatedCost := h.billingService.EstimateCost(req.Model, estimatedTokens)

	log.Printf("[%s] Request: model=%s, stream=%v, estimated_tokens=%d, estimated_cost=%.6f",
		reqID, req.Model, req.Stream, estimatedTokens, estimatedCost)

	if apiKey == nil {
		c.JSON(http.StatusUnauthorized, model.NewAPIError("authentication_error", "API key required"))
		return
	}

	if !model.KeyAllowsModel(apiKey, req.Model) {
		h.appendChatAudit(c, apiKey, req.Model, req.Stream, "model_denied", startAt, reqID)
		c.JSON(http.StatusForbidden, model.NewAPIError("invalid_request_error", "Model not allowed for this API key"))
		return
	}

	if err := h.subService.EnforceBeforeRequest(c.Request.Context(), apiKey.UserID, req.Model, estimatedCost); err != nil {
		h.appendChatAudit(c, apiKey, req.Model, req.Stream, "subscription_denied", startAt, reqID)
		if h.respondSubscriptionError(c, err) {
			return
		}
	}

	// Pre-deduct from user account (optional per-key spend limit)
	if err := h.billingService.PreDeductForAPI(c.Request.Context(), apiKey, estimatedCost); err != nil {
		if err == store.ErrInsufficientBalance || err == store.ErrKeySpendLimitExceeded {
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "insufficient_balance", startAt, reqID)
			msg := "Insufficient account balance"
			if err == store.ErrKeySpendLimitExceeded {
				msg = "API key spend limit exceeded"
			}
			c.JSON(http.StatusPaymentRequired, model.NewAPIError("insufficient_balance", msg))
		} else {
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "billing_error", startAt, reqID)
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to process billing"))
		}
		return
	}

	if model.ModelKindOf(req.Model) == model.ModelKindImage {
		h.handleImageForAPI(c, &req, apiKey, estimatedCost, reqID, startAt)
		return
	}

	// Route to provider
	resp, err := h.providerService.RouteRequest(c.Request.Context(), &req)
	if err != nil {
		// Refund on failure
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, estimatedCost)

		if err == service.ErrNoProviderForModel {
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "client_error", startAt, reqID)
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Model not supported: "+req.Model))
		} else {
			log.Printf("[%s] Provider error: %v", reqID, err)
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "upstream_error", startAt, reqID)
			c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to reach upstream provider"))
		}
		return
	}
	defer resp.Body.Close()

	// Check upstream status
	if resp.StatusCode != http.StatusOK {
		// Refund on upstream error
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, estimatedCost)

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[%s] Upstream error: status=%d, body=%s", reqID, resp.StatusCode, string(body))
		h.appendChatAudit(c, apiKey, req.Model, req.Stream, "upstream_error", startAt, reqID)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	// Handle response based on stream mode
	if req.Stream {
		h.handleStreamResponse(c, resp, req.Model, estimatedCost, reqID, apiKey, startAt)
	} else {
		h.handleNonStreamResponse(c, resp, req.Model, estimatedCost, reqID, apiKey, startAt)
	}
}

// DashboardChatCompletions handles POST /dashboard/chat/completions (JWT, account billing).
func (h *ChatHandler) DashboardChatCompletions(c *gin.Context) {
	var req model.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Invalid request body: "+err.Error()))
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	startAt := time.Now()
	requestID, _ := c.Get("request_id")
	reqID := fmt.Sprintf("%v", requestID)

	estimatedTokens := service.CountInputTokens(req.Messages)
	estimatedCost := h.billingService.EstimateCost(req.Model, estimatedTokens)

	if h.subService.Enabled() {
		if err := h.subService.EnforceBeforeRequest(c.Request.Context(), uid, req.Model, estimatedCost); err != nil {
			if h.respondSubscriptionError(c, err) {
				return
			}
		}
	} else if !h.dashboardModelAllowed(req.Model) {
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Model not enabled for dashboard chat"))
		return
	}

	grantUSD := 0.0
	if h.cfg != nil {
		grantUSD = h.cfg.AccountMonthlyGrantUSD
	}
	if err := h.billingService.PreDeductForChat(c.Request.Context(), uid, estimatedCost, grantUSD); err != nil {
		if err == store.ErrInsufficientBalance {
			c.JSON(http.StatusPaymentRequired, model.NewAPIError("insufficient_balance", "Insufficient account balance"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to process billing"))
		}
		return
	}

	if model.ModelKindOf(req.Model) == model.ModelKindImage {
		h.handleImageForDashboard(c, &req, uid, estimatedCost, reqID)
		return
	}

	resp, err := h.providerService.RouteRequest(c.Request.Context(), &req)
	if err != nil {
		h.billingService.RefundForChat(c.Request.Context(), uid, estimatedCost)
		if err == service.ErrNoProviderForModel {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Model not supported: "+req.Model))
		} else {
			c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to reach upstream provider"))
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.billingService.RefundForChat(c.Request.Context(), uid, estimatedCost)
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	if req.Stream {
		h.handleStreamResponseJWT(c, resp, req.Model, estimatedCost, reqID, uid, startAt)
	} else {
		h.handleNonStreamResponseJWT(c, resp, req.Model, estimatedCost, reqID, uid, startAt)
	}
}

func (h *ChatHandler) handleNonStreamResponseJWT(c *gin.Context, resp *http.Response, modelName string, preDeducted float64, requestID, userID string, startAt time.Time) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.billingService.RefundForChat(c.Request.Context(), userID, preDeducted)
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to read response"))
		return
	}
	var openaiResp *model.ChatCompletionResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		h.billingService.RefundForChat(c.Request.Context(), userID, preDeducted)
		c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to parse upstream response"))
		return
	}
	actualCost := h.billingService.CalculateActualCost(modelName, openaiResp.Usage)
	_ = h.billingService.FinalizeForChat(c.Request.Context(), userID, preDeducted, actualCost, openaiResp.Usage, modelName, requestID)
	_ = h.subService.RecordSpend(c.Request.Context(), userID, actualCost)
	respJSON, _ := json.Marshal(openaiResp)
	c.Data(http.StatusOK, "application/json", respJSON)
}

func (h *ChatHandler) handleStreamResponseJWT(c *gin.Context, resp *http.Response, modelName string, preDeducted float64, requestID, userID string, startAt time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.billingService.RefundForChat(c.Request.Context(), userID, preDeducted)
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Streaming not supported"))
		return
	}
	converter := service.NewStreamConverter("", modelName)
	usage, err := converter.ProcessOpenAIStream(resp.Body, c.Writer, flusher)
	var actualCost float64
	var u model.Usage
	if usage != nil && usage.TotalTokens > 0 {
		u = *usage
		actualCost = h.billingService.CalculateActualCost(modelName, u)
	} else {
		actualCost = preDeducted
		u = model.Usage{TotalTokens: 0}
	}
	_ = h.billingService.FinalizeForChat(c.Request.Context(), userID, preDeducted, actualCost, u, modelName, requestID)
	_ = h.subService.RecordSpend(c.Request.Context(), userID, actualCost)
	_ = err
	_ = startAt
}

func (h *ChatHandler) handleNonStreamResponse(c *gin.Context, resp *http.Response, modelName string, preDeducted float64, requestID string, apiKey *model.APIKey, startAt time.Time) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, preDeducted)
		h.appendChatAudit(c, apiKey, modelName, false, "internal_error", startAt, requestID)
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to read response"))
		return
	}

	// Determine provider from content-type or response format
	contentType := resp.Header.Get("Content-Type")

	var openaiResp *model.ChatCompletionResponse

	if strings.Contains(contentType, "application/json") {
		// Try to parse as Anthropic first
		var anthropicResp service.AnthropicResponse
		if err := json.Unmarshal(body, &anthropicResp); err == nil && anthropicResp.Type == "message" {
			openaiResp = service.ConvertAnthropicResponse(&anthropicResp, modelName)
		} else {
			// Try OpenAI format
			if err := json.Unmarshal(body, &openaiResp); err != nil {
				h.billingService.RefundForAPI(c.Request.Context(), apiKey, preDeducted)
				log.Printf("[%s] Failed to parse response: %v", requestID, err)
				h.appendChatAudit(c, apiKey, modelName, false, "upstream_error", startAt, requestID)
				c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to parse upstream response"))
				return
			}
		}
	}

	if openaiResp == nil {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, preDeducted)
		h.appendChatAudit(c, apiKey, modelName, false, "upstream_error", startAt, requestID)
		c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Unexpected upstream response format"))
		return
	}

	// Calculate actual cost
	actualCost := h.billingService.CalculateActualCost(modelName, openaiResp.Usage)

	// Finalize billing
	if err := h.billingService.FinalizeForAPI(c.Request.Context(), apiKey, preDeducted, actualCost, openaiResp.Usage, modelName, requestID); err != nil {
		log.Printf("[%s] Failed to finalize billing: %v", requestID, err)
	}
	_ = h.subService.RecordSpend(c.Request.Context(), apiKey.UserID, actualCost)

	log.Printf("[%s] Response: tokens=%d (in=%d, out=%d), actual_cost=%.6f",
		requestID, openaiResp.Usage.TotalTokens, openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens, actualCost)

	h.appendChatAudit(c, apiKey, modelName, false, "success", startAt, requestID)

	// Return response
	respJSON, _ := json.Marshal(openaiResp)
	c.Data(http.StatusOK, "application/json", respJSON)
}

func (h *ChatHandler) handleStreamResponse(c *gin.Context, resp *http.Response, modelName string, preDeducted float64, requestID string, apiKey *model.APIKey, startAt time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, preDeducted)
		h.appendChatAudit(c, apiKey, modelName, true, "internal_error", startAt, requestID)
		c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Streaming not supported"))
		return
	}

	// Determine provider and process stream
	contentType := resp.Header.Get("Content-Type")
	converter := service.NewStreamConverter("", modelName)

	var usage *model.Usage
	var err error

	if strings.Contains(contentType, "text/event-stream") {
		// Could be either Anthropic or OpenAI
		// For now, try OpenAI format first (pass-through)
		// TODO: Detect provider from response format
		usage, err = converter.ProcessOpenAIStream(resp.Body, c.Writer, flusher)

		if usage == nil || usage.TotalTokens == 0 {
			// Try Anthropic format
			// Note: This won't work as we've already consumed the stream
			// In production, we need to detect provider before processing
		}
	}

	if err != nil {
		log.Printf("[%s] Stream error: %v", requestID, err)
	}

	// Calculate actual cost (may be estimated if usage not available)
	var actualCost float64
	if usage != nil && usage.TotalTokens > 0 {
		actualCost = h.billingService.CalculateActualCost(modelName, *usage)
	} else {
		// Use pre-deducted as actual (conservative)
		actualCost = preDeducted
		usage = &model.Usage{TotalTokens: int(preDeducted * 1000 / 0.003)} // Rough estimate
	}

	// Finalize billing
	if err := h.billingService.FinalizeForAPI(c.Request.Context(), apiKey, preDeducted, actualCost, *usage, modelName, requestID); err != nil {
		log.Printf("[%s] Failed to finalize billing: %v", requestID, err)
	}
	_ = h.subService.RecordSpend(c.Request.Context(), apiKey.UserID, actualCost)

	if err != nil {
		h.appendChatAudit(c, apiKey, modelName, true, "stream_error", startAt, requestID)
	} else {
		h.appendChatAudit(c, apiKey, modelName, true, "success", startAt, requestID)
	}

	log.Printf("[%s] Stream complete: estimated_cost=%.6f, actual_cost=%.6f", requestID, preDeducted, actualCost)
}
