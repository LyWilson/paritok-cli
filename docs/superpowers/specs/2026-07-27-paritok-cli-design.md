# paritok-cli Design Spec

> **Goal:** Build a cost-optimized, standalone interactive coding agent CLI, selectively porting TUI and file-system modules from the opencode project.

**Architecture:** Single Go binary with Cobra CLI, Charm TUI, and a thin net/http client talking to the Paritok OpenAI-compatible proxy. No provider routing, no model selection menus.

**Tech Stack:** Go 1.26, cobra, bubbletea, lipgloss, glamour, chroma, mark3labs/mcp-go, net/http

---

## 1. Project Structure

```
paritok-cli/
├── main.go                       # Entry point → cmd.Execute()
├── go.mod / go.sum
├── cmd/
│   └── root.go                   # Cobra root + auth subcommand
├── internal/
│   ├── config/
│   │   └── config.go             # $HOME/.paritok.json + env fallback
│   ├── client/
│   │   └── client.go             # Thin streaming HTTP client → Paritok proxy
│   └── mcp/
│       └── mcp.go                # MCP client (stdio transport, tool listing/calling)
├── sample-mcp-config.json        # Neon DB MCP server config for headless testing
└── docs/
    └── superpowers/
        └── specs/                # Design documents
```

## 2. Configuration & Authentication

**File:** `~/.paritok.json`
```json
{ "api_key": "sk-...", "base_url": "http://127.0.0.1:8080" }
```

**Precedence (highest first):**
1. `PARITOK_API_KEY` env var
2. `~/.paritok.json` file
3. `PARITOK_BASE_URL` env var (default: `http://127.0.0.1:8080`)

**Commands:**
- `paritok-cli auth <api_key>` — writes key to `~/.paritok.json` with `0600` permissions
- No provider detection, no model selection

## 3. CLI Entry Point

**Framework:** `spf13/cobra`

- **`paritok-cli`** (no subcommand): Starts the interactive REPL (TUI), placeholder until TUI porting
- **`paritok-cli auth <key>`**: Saves API key
- State flags: `--help`, `--version`

## 4. HTTP Client

**Implementation:** Pure `net/http` — no LLM SDKs

- **Method:** `StreamChat(ctx, model, messages, tools) → chan StreamEvent`
- **Endpoint:** `POST {baseURL}/v1/chat/completions`
- **Auth header:** `Authorization: Bearer {apiKey}`
- **Request body (OpenAI-compatible):**
  ```json
  { "model": "paritok/default", "messages": [...], "stream": true, "tools": [...] }
  ```
- **Response parsing:** SSE via `bufio.Scanner` reading `data:` lines
- **Stream events:** TextDelta, ToolCall, Done, Error

## 5. MCP Integration

**Library:** `mark3labs/mcp-go`

- **Transport:** stdio (child process)
- **Lifecycle:** Initialize → ListTools → (CallTool) → Close
- **Config format:** Matches opencode's `mcpServers` JSON schema
- **Tool naming:** `{serverName}_{toolName}` to avoid collisions

## 6. TUI (Next Phase)

Placeholder until TUI porting in the next phase. The Charm depedencies (bubbletea, lipgloss, glamour, chroma) are already in go.mod ready for porting.

## 7. Non-Goals (Explicitly Excluded)

- No provider matrix or model selection
- No database/SQLite layer
- No LSP integration
- No session persistence beyond what the Paritok proxy provides
- No GitHub Copilot token detection
