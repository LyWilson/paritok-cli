package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LyWilson/prompt-diet-coach/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolDefinition struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function,omitempty"`
}

type StreamEvent struct {
	Type      StreamEventType
	Content   string
	ToolCall  *ToolCall
	Done      bool
	Error     error
}

type StreamEventType int

const (
	EventText StreamEventType = iota
	EventToolCall
	EventDone
	EventError
)

type ToolCall struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(cfg *config.Config) *Client {
	key := cfg.NvidiaAPIKey
	if key == "" {
		key = cfg.APIKey
	}
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  key,
		http:    &http.Client{},
	}
}

type chatRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	Stream      bool            `json:"stream"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Choices []choice `json:"choices"`
}

type choice struct {
	Index        int     `json:"index"`
	Delta        delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type delta struct {
	Role             string    `json:"role,omitempty"`
	Content          string    `json:"content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

func (c *Client) StreamChat(ctx context.Context, model string, messages []Message, tools []ToolDefinition) (<-chan StreamEvent, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("api key not set — run 'paritok-cli auth-nvidia <key>' or set NVIDIA_API_KEY")
	}

	body := chatRequest{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
	}
	if len(tools) > 0 {
		body.Tools = tools
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	events := make(chan StreamEvent, 64)

	go func() {
		defer resp.Body.Close()
		defer close(events)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				events <- StreamEvent{Type: EventDone, Done: true}
				return
			}

			var chunk chatResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			ch := chunk.Choices[0]

			if ch.Delta.Content != "" {
				events <- StreamEvent{Type: EventText, Content: ch.Delta.Content}
			}

			for _, tc := range ch.Delta.ToolCalls {
				events <- StreamEvent{Type: EventToolCall, ToolCall: &tc}
			}

			if ch.FinishReason != nil {
				events <- StreamEvent{Type: EventDone, Done: true}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			events <- StreamEvent{Type: EventError, Error: fmt.Errorf("read stream: %w", err)}
		}
	}()

	return events, nil
}
