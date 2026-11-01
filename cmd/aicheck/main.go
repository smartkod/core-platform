// Command aicheck performs one chat completion against the configured provider.
//
// It exists so the AI wiring can be verified without starting the server: run it
// after changing AI_* settings to confirm the endpoint, key and model actually
// work together. It prints usage and latency, which is also the quickest way to
// catch a provider that is reachable but slow.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"core-platform/platform/ai"
)

func main() {
	baseURL := os.Getenv("AI_BASE_URL")
	apiKey := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_DEFAULT_MODEL")

	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "aicheck: AI_BASE_URL is not set")
		os.Exit(2)
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "aicheck: AI_API_KEY is not set")
		os.Exit(2)
	}

	provider, err := ai.NewOpenAI(ai.Config{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		DefaultModel: model,
		Timeout:      60 * time.Second,
		MaxRetries:   1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aicheck: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("provider : %s\n", provider.Name())
	fmt.Printf("endpoint : %s\n", baseURL)
	fmt.Printf("model    : %s\n", model)
	fmt.Println("sending one request...")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: "Reply with exactly: OK"},
		},
		MaxTokens: 32,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nOK")
	fmt.Printf("  reply      : %q\n", res.Content)
	fmt.Printf("  served by  : %s\n", res.Model)
	fmt.Printf("  tokens     : prompt=%d completion=%d total=%d\n",
		res.Usage.PromptTokens, res.Usage.CompletionTokens, res.Usage.TotalTokens)
	fmt.Printf("  latency    : %s\n", res.Latency.Round(time.Millisecond))
}
