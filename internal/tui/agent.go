package tui

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yourusername/paritok-cli/internal/client"
	"github.com/yourusername/paritok-cli/internal/tools"
)

const maxRounds = 20

type agentMsg struct {
	kind     string // "text", "tool_call", "tool_result", "done", "error"
	content  string
	toolName string
	toolID   string
	messages []client.Message // set on "done" — final conversation state
}

func runAgent(cl *client.Client, model string, initialMessages []client.Message, toolDefs []client.ToolDefinition, ch chan<- agentMsg) {
	messages := make([]client.Message, len(initialMessages))
	copy(messages, initialMessages)

	for round := 0; round < maxRounds; round++ {
		ctx := context.Background()
		events, err := cl.StreamChat(ctx, model, messages, toolDefs)
		if err != nil {
			ch <- agentMsg{kind: "error", content: err.Error()}
			return
		}

		var textBuf strings.Builder
		var toolCalls []client.ToolCall

		for evt := range events {
			switch evt.Type {
			case client.EventText:
				textBuf.WriteString(evt.Content)
				ch <- agentMsg{kind: "text", content: evt.Content}
			case client.EventToolCall:
				toolCalls = append(toolCalls, *evt.ToolCall)
			case client.EventDone:
			case client.EventError:
				ch <- agentMsg{kind: "error", content: evt.Error.Error()}
				return
			}
		}

		if len(toolCalls) == 0 {
			messages = append(messages, client.Message{Role: "assistant", Content: textBuf.String()})
			ch <- agentMsg{kind: "done", messages: messages}
			return
		}

		messages = append(messages, client.Message{
			Role:      "assistant",
			Content:   textBuf.String(),
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			ch <- agentMsg{kind: "tool_call", toolName: tc.Function.Name, toolID: tc.ID, content: tc.Function.Arguments}
			result, err := tools.Dispatch(tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				result = "Error: " + err.Error()
			}
			messages = append(messages, client.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
			ch <- agentMsg{kind: "tool_result", toolID: tc.ID, content: result}
		}
	}

	ch <- agentMsg{kind: "error", content: "agent loop exceeded max rounds"}
}
