package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sub2api-go/internal/model"
	"sub2api-go/internal/service"
)

func (h *ChatHandler) dashboardModelAllowed(modelName string) bool {
	if h.subService.Enabled() {
		return true
	}
	if h.cfg == nil || len(h.cfg.ChatEnabledModels) == 0 {
		return true
	}
	return model.ModelInAllowlist(modelName, h.cfg.ChatEnabledModels)
}

func (h *ChatHandler) handleImageForAPI(
	c *gin.Context,
	req *model.ChatCompletionRequest,
	apiKey *model.APIKey,
	estimatedCost float64,
	reqID string,
	startAt time.Time,
) {
	if req.Stream {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, estimatedCost)
		h.appendChatAudit(c, apiKey, req.Model, true, "client_error", startAt, reqID)
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image generation does not support streaming"))
		return
	}

	prompt := model.LastUserMessageContent(req.Messages)
	if prompt == "" {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, estimatedCost)
		h.appendChatAudit(c, apiKey, req.Model, false, "client_error", startAt, reqID)
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image generation requires a user prompt"))
		return
	}

	imageURL, err := h.providerService.GenerateImage(c.Request.Context(), req.Model, prompt)
	if err != nil {
		h.billingService.RefundForAPI(c.Request.Context(), apiKey, estimatedCost)
		h.appendChatAudit(c, apiKey, req.Model, false, "upstream_error", startAt, reqID)
		if err == service.ErrNoProviderForModel {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image model not configured"))
			return
		}
		c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to generate image"))
		return
	}

	actualCost := h.billingService.CalculateActualCost(req.Model, model.Usage{})
	usage := model.Usage{PromptTokens: service.CountInputTokens(req.Messages), CompletionTokens: 0, TotalTokens: service.CountInputTokens(req.Messages)}
	_ = h.billingService.FinalizeForAPI(c.Request.Context(), apiKey, estimatedCost, actualCost, usage, req.Model, reqID)
	_ = h.subService.RecordSpend(c.Request.Context(), apiKey.UserID, actualCost)
	h.appendChatAudit(c, apiKey, req.Model, false, "success", startAt, reqID)

	content := fmt.Sprintf("Generated image:\n%s", imageURL)
	resp := model.ChatCompletionResponse{
		ID:      "chatcmpl-image-" + reqID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.Choice{{
			Index: 0,
			Message: &model.Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: "stop",
		}},
		Usage: usage,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) handleImageForDashboard(
	c *gin.Context,
	req *model.ChatCompletionRequest,
	userID string,
	estimatedCost float64,
	reqID string,
) {
	if req.Stream {
		h.billingService.RefundForChat(c.Request.Context(), userID, estimatedCost)
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image generation does not support streaming"))
		return
	}

	prompt := model.LastUserMessageContent(req.Messages)
	if prompt == "" {
		h.billingService.RefundForChat(c.Request.Context(), userID, estimatedCost)
		c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image generation requires a user prompt"))
		return
	}

	imageURL, err := h.providerService.GenerateImage(c.Request.Context(), req.Model, prompt)
	if err != nil {
		h.billingService.RefundForChat(c.Request.Context(), userID, estimatedCost)
		if err == service.ErrNoProviderForModel {
			c.JSON(http.StatusBadRequest, model.NewAPIError("invalid_request_error", "Image model not configured"))
			return
		}
		c.JSON(http.StatusBadGateway, model.NewAPIError("upstream_error", "Failed to generate image"))
		return
	}

	actualCost := h.billingService.CalculateActualCost(req.Model, model.Usage{})
	usage := model.Usage{PromptTokens: service.CountInputTokens(req.Messages), TotalTokens: service.CountInputTokens(req.Messages)}
	_ = h.billingService.FinalizeForChat(c.Request.Context(), userID, estimatedCost, actualCost, usage, req.Model, reqID)
	_ = h.subService.RecordSpend(c.Request.Context(), userID, actualCost)

	content := fmt.Sprintf("Generated image:\n%s", imageURL)
	resp := model.ChatCompletionResponse{
		ID:      "chatcmpl-image-" + reqID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.Choice{{
			Index: 0,
			Message: &model.Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: "stop",
		}},
		Usage: usage,
	}
	c.JSON(http.StatusOK, resp)
}
