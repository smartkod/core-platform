// Package ai provides an OpenAI-compatible chat client.
//
// It is a transport, not a feature: no quota, no caching, no prompt templates.
// Those belong to the L3 module, which will use this package. Keeping the HTTP
// details here means the module can be tested without a network and the endpoint
// can be swapped by configuration alone.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Role identifies who produced a message.
type Role string

// Chat roles defined by the OpenAI schema.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest asks for a completion.
type ChatRequest struct {
	Messages    []Message
	Model       string
	MaxTokens   int
	Temperature float64
	// Stop ends generation at any of these strings.
	Stop []string
}

// Usage reports token consumption. It is the basis of cost tracking.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResult is a completed response.
type ChatResult struct {
	Content string
	Model   string
	Usage   Usage
	// Latency is the wall time of the request, useful for spotting a slow
	// provider before a user complains.
	Latency time.Duration
}

// Provider is the contract the L3 module will depend on.
//
// It is deliberately one method. Streaming, embeddings and tool calls are absent
// because they are absent from V1 (L3.md 6); adding a method later is a minor
// version change, removing one would be major.
type Provider interface {
	// Name identifies the provider in logs and usage records.
	Name() string
	// Chat performs one completion.
	Chat(ctx context.Context, req ChatRequest) (ChatResult, error)
}

// Config configures the OpenAI-compatible adapter.
type Config struct {
	// BaseURL is the API root, e.g. "https://host/v1".
	BaseURL string
	// APIKey is a bearer token. It comes from the environment.
	APIKey string
	// DefaultModel is used when a request omits one.
	DefaultModel string
	// Timeout bounds one request.
	Timeout time.Duration
	// MaxRetries applies to 5xx and network failures only, never to 4xx: a bad
	// request must not be repeated.
	MaxRetries int
	// HTTPClient lets a test inject a client. Nil means a default is built.
	HTTPClient *http.Client
}

// OpenAI talks to any OpenAI-compatible chat completions endpoint.
//
// The name is the protocol, not the vendor: the same adapter serves OpenAI,
// Azure, and any self-hosted gateway that mirrors the schema.
type OpenAI struct {
	cfg    Config
	client *http.Client
}

// NewOpenAI builds the adapter.
func NewOpenAI(cfg Config) (*OpenAI, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("ai: base URL is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}

	return &OpenAI{cfg: cfg, client: client}, nil
}

// Name implements Provider.
func (o *OpenAI) Name() string { return "openai-compatible" }

// wireRequest is the JSON body the endpoint expects.
type wireRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
	Stream      bool      `json:"stream"`
}

// wireResponse is the subset of the response body this package reads.
type wireResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Chat performs one completion.
func (o *OpenAI) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	if len(req.Messages) == 0 {
		return ChatResult{}, fmt.Errorf("ai: at least one message is required")
	}

	model := req.Model
	if model == "" {
		model = o.cfg.DefaultModel
	}
	if model == "" {
		return ChatResult{}, fmt.Errorf("ai: no model given and no default configured")
	}

	body, err := json.Marshal(wireRequest{
		Model:       model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stop:        req.Stop,
	})
	if err != nil {
		return ChatResult{}, fmt.Errorf("ai: encode request: %w", err)
	}

	start := time.Now()
	var lastErr error

	// Retry loop. attempt 0 is the first try, so the total attempts are
	// MaxRetries+1.
	for attempt := 0; attempt <= o.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Linear backoff is enough here: the L3 module adds jitter when it
			// owns the retry policy across providers. Repeating a request that
			// already failed fast keeps latency predictable.
			select {
			case <-ctx.Done():
				return ChatResult{}, ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}

		result, retryable, err := o.do(ctx, body, start)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable {
			return ChatResult{}, err
		}
	}

	return ChatResult{}, fmt.Errorf("ai: giving up after %d attempts: %w",
		o.cfg.MaxRetries+1, lastErr)
}

// do performs a single HTTP attempt and reports whether it is worth retrying.
func (o *OpenAI) do(ctx context.Context, body []byte, start time.Time) (ChatResult, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, false, fmt.Errorf("ai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		// A network error is retryable; cancelling the context is not.
		if ctx.Err() != nil {
			return ChatResult{}, false, ctx.Err()
		}
		return ChatResult{}, true, fmt.Errorf("ai: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Cap the read so a misbehaving endpoint cannot exhaust memory.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return ChatResult{}, true, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// The body carries the provider's reason; the message is surfaced because
		// it is about the caller's request, not our internals.
		var wr wireResponse
		msg := strings.TrimSpace(string(raw))
		if json.Unmarshal(raw, &wr) == nil && wr.Error != nil && wr.Error.Message != "" {
			msg = wr.Error.Message
		}
		retryable := resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
		return ChatResult{}, retryable, &HTTPError{Status: resp.StatusCode, Message: msg}
	}

	var wr wireResponse
	if err := json.Unmarshal(raw, &wr); err != nil {
		return ChatResult{}, false, fmt.Errorf("ai: decode response: %w", err)
	}
	if wr.Error != nil {
		return ChatResult{}, false, &HTTPError{Status: resp.StatusCode, Message: wr.Error.Message}
	}
	if len(wr.Choices) == 0 {
		return ChatResult{}, false, fmt.Errorf("ai: response contained no choices")
	}

	return ChatResult{
		Content: wr.Choices[0].Message.Content,
		Model:   wr.Model,
		Usage:   wr.Usage,
		Latency: time.Since(start),
	}, false, nil
}

// HTTPError is a provider-side rejection.
//
// It carries the status so a caller can tell "you sent a bad request" (4xx) from
// "the provider is unwell" (5xx) without parsing strings.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("ai: provider returned %d: %s", e.Status, e.Message)
}

// Compile-time check.
var _ Provider = (*OpenAI)(nil)
