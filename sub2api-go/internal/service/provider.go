package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sub2api-go/internal/config"
	"sub2api-go/internal/model"
)

var (
	ErrNoProviderForModel = errors.New("no provider available for model")
	ErrUpstreamError      = errors.New("upstream provider error")
)

// ProviderService handles routing requests to upstream providers
type ProviderService struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewProviderService(cfg *config.Config) *ProviderService {
	return &ProviderService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// RouteRequest routes a chat completion request to the appropriate provider
func (s *ProviderService) RouteRequest(ctx context.Context, req *model.ChatCompletionRequest) (*http.Response, error) {
	_, upstream, ok := model.ResolvePlatformModel(req.Model)
	if !ok {
		upstream = req.Model
	}
	routeReq := *req
	routeReq.Model = upstream

	provider := s.findProviderForModel(upstream)
	if provider == nil {
		return nil, ErrNoProviderForModel
	}

	// Convert request to provider-specific format
	switch provider.Name {
	case "anthropic":
		return s.callAnthropic(ctx, provider, &routeReq)
	case "openai", "deepseek", "google":
		return s.callOpenAI(ctx, provider, &routeReq)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider.Name)
	}
}

// GenerateImage calls OpenAI-compatible images API and returns the first image URL.
func (s *ProviderService) GenerateImage(ctx context.Context, platformModelID, prompt string) (string, error) {
	_, upstream, ok := model.ResolvePlatformModel(platformModelID)
	if !ok {
		upstream = platformModelID
	}
	provider := s.findProviderForModel(upstream)
	if provider == nil {
		return "", ErrNoProviderForModel
	}

	body, err := json.Marshal(map[string]interface{}{
		"model":  upstream,
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.BaseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status=%d body=%s", ErrUpstreamError, resp.StatusCode, string(raw))
	}

	var parsed struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Data) == 0 || parsed.Data[0].URL == "" {
		return "", fmt.Errorf("%w: empty image response", ErrUpstreamError)
	}
	return parsed.Data[0].URL, nil
}

func (s *ProviderService) findProviderForModel(modelName string) *config.ProviderConfig {
	for i := range s.cfg.Providers {
		p := &s.cfg.Providers[i]
		for _, m := range p.Models {
			if m == modelName || strings.HasPrefix(modelName, strings.TrimSuffix(m, "*")) {
				return p
			}
		}
	}
	return nil
}

// ==================== Anthropic ====================

type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []AnthropicMessage `json:"messages"`
	Stream      bool               `json:"stream,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Role         string           `json:"role"`
	Content      []ContentBlock   `json:"content"`
	Model        string           `json:"model"`
	StopReason   string           `json:"stop_reason"`
	StopSequence string           `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage   `json:"usage"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (s *ProviderService) callAnthropic(ctx context.Context, provider *config.ProviderConfig, req *model.ChatCompletionRequest) (*http.Response, error) {
	// Convert messages
	var messages []AnthropicMessage
	for _, m := range req.Messages {
		// Skip system messages for now, Anthropic handles them differently
		if m.Role == "system" {
			continue
		}
		messages = append(messages, AnthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	anthropicReq := AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   maxTokens,
		Messages:    messages,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", provider.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	return s.httpClient.Do(httpReq)
}

// ConvertAnthropicResponse converts Anthropic response to OpenAI format
func ConvertAnthropicResponse(anthropicResp *AnthropicResponse, requestModel string) *model.ChatCompletionResponse {
	var content string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &model.ChatCompletionResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []model.Choice{
			{
				Index: 0,
				Message: &model.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: convertFinishReason(anthropicResp.StopReason),
			},
		},
		Usage: model.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}
}

func convertFinishReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return reason
	}
}

// ==================== OpenAI / DeepSeek ====================

func (s *ProviderService) callOpenAI(ctx context.Context, provider *config.ProviderConfig, req *model.ChatCompletionRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	return s.httpClient.Do(httpReq)
}

// ==================== Stream Processing ====================

// StreamConverter converts provider-specific stream to OpenAI format
type StreamConverter struct {
	provider string
	model    string
}

func NewStreamConverter(provider, model string) *StreamConverter {
	return &StreamConverter{provider: provider, model: model}
}

// ProcessAnthropicStream reads Anthropic SSE and converts to OpenAI format
func (c *StreamConverter) ProcessAnthropicStream(reader io.Reader, writer io.Writer, flusher http.Flusher) (*model.Usage, error) {
	scanner := bufio.NewScanner(reader)
	var usage model.Usage
	var totalContent string

	for scanner.Scan() {
		line := scanner.Text()
		
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			fmt.Fprintf(writer, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "content_block_delta":
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					totalContent += text
					chunk := model.StreamChunk{
						ID:      "chatcmpl-stream",
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   c.model,
						Choices: []model.Choice{
							{
								Index: 0,
								Delta: &model.Delta{Content: text},
							},
						},
					}
					chunkJSON, _ := json.Marshal(chunk)
					fmt.Fprintf(writer, "data: %s\n\n", chunkJSON)
					flusher.Flush()
				}
			}

		case "message_delta":
			if usageData, ok := event["usage"].(map[string]interface{}); ok {
				if outputTokens, ok := usageData["output_tokens"].(float64); ok {
					usage.CompletionTokens = int(outputTokens)
				}
			}

		case "message_start":
			if message, ok := event["message"].(map[string]interface{}); ok {
				if usageData, ok := message["usage"].(map[string]interface{}); ok {
					if inputTokens, ok := usageData["input_tokens"].(float64); ok {
						usage.PromptTokens = int(inputTokens)
					}
				}
			}

		case "message_stop":
			// Final chunk with finish_reason
			chunk := model.StreamChunk{
				ID:      "chatcmpl-stream",
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   c.model,
				Choices: []model.Choice{
					{
						Index:        0,
						Delta:        &model.Delta{},
						FinishReason: "stop",
					},
				},
			}
			chunkJSON, _ := json.Marshal(chunk)
			fmt.Fprintf(writer, "data: %s\n\n", chunkJSON)
			flusher.Flush()
		}
	}

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return &usage, scanner.Err()
}

// ProcessOpenAIStream passes through OpenAI format stream and extracts usage
func (c *StreamConverter) ProcessOpenAIStream(reader io.Reader, writer io.Writer, flusher http.Flusher) (*model.Usage, error) {
	scanner := bufio.NewScanner(reader)
	var usage model.Usage

	for scanner.Scan() {
		line := scanner.Text()
		
		// Pass through directly
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			
			if data != "[DONE]" {
				var chunk model.StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err == nil {
					if chunk.Usage != nil {
						usage = *chunk.Usage
					}
				}
			}
			
			fmt.Fprintf(writer, "%s\n\n", line)
			flusher.Flush()
		}
	}

	return &usage, scanner.Err()
}
