package clients

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"time"
)

type OpenAIClient struct {
apiKey  string
baseURL string
client  *http.Client
}

func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
return &OpenAIClient{
apiKey:  apiKey,
baseURL: baseURL,
client: &http.Client{
Timeout: 30 * time.Second,
},
}
}

type OpenAIRequest struct {
Model       string         json:"model"
Messages    []OpenAIMessage json:"messages"
MaxTokens   int            json:"max_tokens,omitempty"
Temperature float64        json:"temperature,omitempty"
}

type OpenAIMessage struct {
Role    string json:"role"
Content string json:"content"
}

type OpenAIResponse struct {
ID      string json:"id"
Choices []struct {
Message OpenAIMessage json:"message"
} json:"choices"
Usage struct {
PromptTokens     int json:"prompt_tokens"
CompletionTokens int json:"completion_tokens"
TotalTokens      int json:"total_tokens"
} json:"usage"
}

func (c *OpenAIClient) Chat(prompt string, model string, maxTokens int, temperature float64) (string, error) {
reqBody := OpenAIRequest{
Model: model,
Messages: []OpenAIMessage{
{Role: "user", Content: prompt},
},
MaxTokens:   maxTokens,
Temperature: temperature,
}

jsonData, err := json.Marshal(reqBody)
if err != nil {
return "", err
}

url := c.baseURL + "/v1/chat/completions"
req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
if err != nil {
return "", err
}

req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+c.apiKey)

resp, err := c.client.Do(req)
if err != nil {
return "", err
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
return "", err
}

if resp.StatusCode != http.StatusOK {
return "", fmt.Errorf("OpenAI API error: %s", string(body))
}

var openAIResp OpenAIResponse
if err := json.Unmarshal(body, &openAIResp); err != nil {
return "", err
}

if len(openAIResp.Choices) == 0 {
return "", fmt.Errorf("no response from OpenAI")
}

return openAIResp.Choices[0].Message.Content, nil
}
