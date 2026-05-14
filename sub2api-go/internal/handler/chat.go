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
	store           store.Store
	cfg             *config.Config
}

func NewChatHandler(ps *service.ProviderService, bs *service.BillingService, s store.Store, cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		providerService: ps,
		billingService:  bs,
		store:           s,
		cfg:             cfg,
	}
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

	keyHash := middleware.GetKeyHash(c)
	requestID, _ := c.Get("request_id")
	reqID := fmt.Sprintf("%v", requestID)

	// Estimate cost and pre-deduct
	estimatedTokens := service.CountInputTokens(req.Messages)
	estimatedCost := h.billingService.EstimateCost(req.Model, estimatedTokens)

	log.Printf("[%s] Request: model=%s, stream=%v, estimated_tokens=%d, estimated_cost=%.6f",
		reqID, req.Model, req.Stream, estimatedTokens, estimatedCost)

	// Pre-deduct
	if err := h.billingService.PreDeduct(c.Request.Context(), keyHash, estimatedCost); err != nil {
		if err == store.ErrInsufficientBalance {
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "insufficient_balance", startAt, reqID)
			c.JSON(http.StatusPaymentRequired, model.NewAPIError("insufficient_balance", "Insufficient balance"))
		} else {
			h.appendChatAudit(c, apiKey, req.Model, req.Stream, "billing_error", startAt, reqID)
			c.JSON(http.StatusInternalServerError, model.NewAPIError("internal_error", "Failed to process billing"))
		}
		return
	}

	// Route to provider
	resp, err := h.providerService.RouteRequest(c.Request.Context(), &req)
	if err != nil {
		// Refund on failure
		h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, estimatedCost)

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
		h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, estimatedCost)

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[%s] Upstream error: status=%d, body=%s", reqID, resp.StatusCode, string(body))
		h.appendChatAudit(c, apiKey, req.Model, req.Stream, "upstream_error", startAt, reqID)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	// Handle response based on stream mode
	if req.Stream {
		h.handleStreamResponse(c, resp, req.Model, keyHash, estimatedCost, reqID, apiKey, startAt)
	} else {
		h.handleNonStreamResponse(c, resp, req.Model, keyHash, estimatedCost, reqID, apiKey, startAt)
	}
}

func (h *ChatHandler) handleNonStreamResponse(c *gin.Context, resp *http.Response, modelName, keyHash string, preDeducted float64, requestID string, apiKey *model.APIKey, startAt time.Time) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, preDeducted)
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
				h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, preDeducted)
				log.Printf("[%s] Failed to parse response: %v", requestID, err)
				h.appendChatAudit(c, apiKey, modelName, false, "upstream_error", startAt, requestID)
				c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to parse upstream response"))
				return
			}
		}
	}

	if openaiResp == nil {
		h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, preDeducted)
		h.appendChatAudit(c, apiKey, modelName, false, "upstream_error", startAt, requestID)
		c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Unexpected upstream response format"))
		return
	}

	// Calculate actual cost
	actualCost := h.billingService.CalculateActualCost(modelName, openaiResp.Usage)

	// Finalize billing
	if err := h.billingService.FinalizeDeduct(c.Request.Context(), keyHash, preDeducted, actualCost, openaiResp.Usage, modelName, requestID); err != nil {
		log.Printf("[%s] Failed to finalize billing: %v", requestID, err)
	}

	log.Printf("[%s] Response: tokens=%d (in=%d, out=%d), actual_cost=%.6f",
		requestID, openaiResp.Usage.TotalTokens, openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens, actualCost)

	h.appendChatAudit(c, apiKey, modelName, false, "success", startAt, requestID)

	// Return response
	respJSON, _ := json.Marshal(openaiResp)
	c.Data(http.StatusOK, "application/json", respJSON)
}

func (h *ChatHandler) handleStreamResponse(c *gin.Context, resp *http.Response, modelName, keyHash string, preDeducted float64, requestID string, apiKey *model.APIKey, startAt time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.billingService.RefundPreDeduct(c.Request.Context(), keyHash, preDeducted)
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
	if err := h.billingService.FinalizeDeduct(c.Request.Context(), keyHash, preDeducted, actualCost, *usage, modelName, requestID); err != nil {
		log.Printf("[%s] Failed to finalize billing: %v", requestID, err)
	}

	if err != nil {
		h.appendChatAudit(c, apiKey, modelName, true, "stream_error", startAt, requestID)
	} else {
		h.appendChatAudit(c, apiKey, modelName, true, "success", startAt, requestID)
	}

	log.Printf("[%s] Stream complete: estimated_cost=%.6f, actual_cost=%.6f", requestID, preDeducted, actualCost)
}
