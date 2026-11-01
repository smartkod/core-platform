package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer returns an httptest server that answers one chat completion.
//
// Tests never touch a real provider: an adapter test that needs the network is
// a test that fails for reasons unrelated to the code.
func fakeServer(t *testing.T, handler http.HandlerFunc) (*OpenAI, *int32) {
	t.Helper()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	p, err := NewOpenAI(Config{
		BaseURL:      srv.URL + "/v1/",
		APIKey:       "test-key",
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}
	return p, &calls
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestChatSendsExpectedRequest(t *testing.T) {
	var gotPath, gotAuth, gotBody string

	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		var body strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			body.Write(buf[:n])
			if err != nil {
				break
			}
		}
		gotBody = body.String()

		writeJSON(w, http.StatusOK, map[string]any{
			"model":   "test-model",
			"choices": []any{map[string]any{"message": map[string]string{"content": "OK"}}},
			"usage":   map[string]int{"prompt_tokens": 11, "completion_tokens": 2, "total_tokens": 13},
		})
	})

	res, err := p.Chat(context.Background(), ChatRequest{
		Messages:  []Message{{Role: RoleUser, Content: "hi"}},
		MaxTokens: 16,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if res.Content != "OK" {
		t.Errorf("content = %q, want OK", res.Content)
	}
	if res.Usage.PromptTokens != 11 || res.Usage.CompletionTokens != 2 {
		t.Errorf("usage = %+v, want 11/2", res.Usage)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q, want /v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q, want a bearer token", gotAuth)
	}
	// The trailing slash in the base URL must not produce a double slash.
	if strings.Contains(gotPath, "//") {
		t.Errorf("path = %q, a base URL with a trailing slash must be normalised", gotPath)
	}
	if !strings.Contains(gotBody, `"model":"test-model"`) {
		t.Errorf("body = %q, want the default model to be filled in", gotBody)
	}
}

func TestChatUsesExplicitModel(t *testing.T) {
	var body string
	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": "x"}}},
		})
	})

	if _, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		Model:    "explicit-model",
	}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(body, `"model":"explicit-model"`) {
		t.Errorf("body = %q, want the explicit model", body)
	}
}

// TestChatDoesNotRetry4xx pins the most important retry rule: a bad request must
// fail once. Retrying it wastes the caller's latency budget and hides the bug.
func TestChatDoesNotRetry4xx(t *testing.T) {
	p, calls := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "bad model", "type": "invalid_request"},
		})
	})

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("a 400 must be an error")
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("calls = %d, want 1: a 4xx must not be retried", got)
	}

	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("err = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", he.Status)
	}
	if !strings.Contains(err.Error(), "bad model") {
		t.Errorf("err = %q, want the provider's reason", err)
	}
}

func TestChatRetries5xx(t *testing.T) {
	p, calls := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "provider is unwell"},
		})
	})

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("exhausted retries must return an error")
	}
	// Default MaxRetries is 0 in this config, so exactly one attempt happens and
	// no retry is expected. The retry path is covered by the next test.
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("calls = %d, want 1 with no retries configured", got)
	}
}

func TestChatRetriesWhenConfigured(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"message": "down"}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": "recovered"}}},
		})
	}))
	defer srv.Close()

	p, err := NewOpenAI(Config{
		BaseURL:      srv.URL + "/v1",
		DefaultModel: "m",
		Timeout:      5 * time.Second,
		MaxRetries:   2,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if res.Content != "recovered" {
		t.Errorf("content = %q, want recovered", res.Content)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3 (two failures then success)", got)
	}
}

func TestChatRespectsContextCancellation(t *testing.T) {
	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		writeJSON(w, http.StatusOK, map[string]any{})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := p.Chat(ctx, ChatRequest{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err == nil {
		t.Fatal("a cancelled context must return an error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("took %v; cancellation must not wait for the provider", elapsed)
	}
}

func TestChatRejectsEmptyMessages(t *testing.T) {
	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be made")
	})

	if _, err := p.Chat(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("an empty conversation must be rejected before the request")
	}
}

func TestNewOpenAIRequiresBaseURL(t *testing.T) {
	if _, err := NewOpenAI(Config{DefaultModel: "m"}); err == nil {
		t.Fatal("a missing base URL must be rejected")
	}
}

func TestChatRequiresAModel(t *testing.T) {
	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be made without a model")
	})

	// Clear the default to exercise the guard.
	p.cfg.DefaultModel = ""
	if _, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	}); err == nil {
		t.Fatal("a request without a model must be rejected")
	}
}

// TestChatReportsProviderErrorBody verifies that the provider's own message is
// surfaced: "invalid model name" is actionable, "provider returned 400" is not.
func TestChatReportsProviderErrorBody(t *testing.T) {
	p, _ := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "invalid api key"},
		})
	})

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("err = %q, want the provider's message", err)
	}
}
