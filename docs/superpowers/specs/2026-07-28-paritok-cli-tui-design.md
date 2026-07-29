# paritok-cli TUI — Interactive Chat Mode

## Goal
A minimal terminal UI that lets the user chat with AI models through the paritok proxy, with streaming responses.

## Design

### New Commands
- `paritok-cli chat [--model <name>]` — starts the interactive TUI. Default model: `nvidia/llama-3.1-nemotron-70b-instruct`.

### Architecture
```
cmd/chat.go -> loads config, creates client, starts bubbletea program
internal/tui/tui.go -> bubbletea Model with:
  - viewport.Model (scrolling output)
  - textinput.Model (single-line input)
  - []client.Message (conversation history)
  - streaming state + channel for async events
```

### Data Flow
1. User types message, presses Enter
2. User message appended to viewport + history
3. Goroutine calls `client.StreamChat` with full history
4. Each SSE `data:` chunk sent to bubbletea via channel
5. Update loop appends tokens to viewport in real-time
6. On stream complete, full response added to history
7. Ready for next input

### Edge Cases
- **No API key** — show error, fall back to help text
- **Connection error** — show error in viewport, re-enable input
- **Empty input** — ignored
- **Resize** — `tea.WindowSizeMsg` reflows viewport
- **Quit** — `/quit` command or Ctrl+C

### Dependencies
- `charmbracelet/bubbles` — for `viewport` and `textinput` widgets (add)
- `charmbracelet/bubbletea` — already in go.mod
- `charmbracelet/lipgloss` — already in go.mod