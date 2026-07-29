# paritok-cli

A cost-optimized interactive coding agent CLI. Routes requests through the **paritok proxy** for compression, then forwards to NVIDIA's API for AI-powered code assistance.

```
prompt > build a calculator using typescript
AI     → generates code
CLI    → runs it, captures output
AI     → fixes errors, iterates
CLI    → returns working code
```

## How It Works

```
┌──────────────┐     POST /v1/chat/completions     ┌──────────┐     NVIDIA API     ┌─────────┐
│ paritok-cli  │ ──── Authorization: NVIDIA key ──→ │ paritok  │ ────────────────→ │ NVIDIA  │
│  (chat TUI)  │     (streaming, no compression)    │  proxy   │                   │  API    │
└──────────────┘                                    └──────────┘                   └─────────┘
       │                                                   ▲
       │ POST https://www.paritok.com/api/compress          │
       │ (compresses content before sending)                │
       └────────────────────────────────────────────────────┘
```

The CLI sends chat completions through the local paritok proxy (port 8080). The proxy forwards to NVIDIA. Separately, the CLI compresses content via paritok's cloud API before sending to save tokens.

## Prerequisites

- **Go 1.26+** — to build the CLI
- **paritok proxy** — the local forwarding proxy
- **PARITOK_API_KEY** — from [paritok.com](https://paritok.com) (for compression)
- **NVIDIA_API_KEY** — from [NVIDIA API](https://build.nvidia.com/) (for AI inference)

## Installation

### 1. Build the CLI

```powershell
git clone https://github.com/LyWilson/paritok-cli.git
cd paritok-cli
go build -o paritok-cli.exe .
```

### 2. Install the paritok proxy

```powershell
pip install paritok
```

Make sure `paritok.exe` is on your PATH (check with `where paritok`).

### 3. Configure `paritok.yaml`

Edit `paritok.yaml` in the project directory:

```yaml
use_gpu_server: true
gpu_server:
  api_key: "pk_live_<your-paritok-api-key>"
```

### 4. Save your API keys

```powershell
paritok-cli auth pk_live_<your-paritok-key>
paritok-cli auth-nvidia nvapi-<your-nvidia-key>
```

Keys are stored in `~/.paritok.json` with `0600` permissions.

## Usage

### Quick start

```powershell
# Terminal 1: start the proxy
paritok-cli proxy start

# Terminal 2: chat
paritok-cli chat

# When done
paritok-cli proxy stop
```

### Commands

| Command | What it does |
|---|---|
| `chat` | Interactive TUI — type messages, AI streams responses |
| `code <prompt>` | Headless coding agent — generates code, runs it, iterates to fix errors |
| `proxy start` | Starts the paritok proxy on port 8080 |
| `proxy stop` | Stops the proxy |
| `auth <key>` | Save paritok API key to `~/.paritok.json` |
| `auth-nvidia <key>` | Save NVIDIA API key to `~/.paritok.json` |

### TUI chat commands

| Command | What it does |
|---|---|
| `/help` | Show available commands |
| `/quit` / `/exit` | Exit the TUI |
| `/code <prompt>` | Enter code iteration mode inside the chat |

### Coding agent

**Headless mode:**
```powershell
paritok-cli code "build a calculator using typescript"
```

This generates code, saves it to a temp file, runs it (e.g. `npx tsx` for TypeScript), captures output, and feeds errors back to the AI to fix. Loops up to 5 iterations.

**In the TUI:**
```
/code build a calculator in python
```

Same loop but progress is shown inline in the chat viewport.

**Supported languages:**

| Language | Runner |
|---|---|
| TypeScript / `.ts` | `npx --yes tsx <file>` |
| JavaScript / `.js` | `node <file>` |
| Python / `.py` | `python <file>` |
| Go / `.go` | `go run <file>` |
| Bash / `.sh` | `bash <file>` |

### Model selection

Default model: `openai/gpt-oss-20b`

```powershell
paritok-cli chat --model "meta/llama-3.1-70b-instruct"
paritok-cli code --model "meta/llama-3.1-70b-instruct" "build a calculator"
```

## Configuration

All config is in `~/.paritok.json`:

```json
{
  "api_key": "pk_live_...",
  "nvidia_api_key": "nvapi-...",
  "base_url": "http://127.0.0.1:8080"
}
```

| Setting | Env var | Default |
|---|---|---|
| `api_key` | `PARITOK_API_KEY` | — |
| `nvidia_api_key` | `NVIDIA_API_KEY` | — |
| `base_url` | `PARITOK_BASE_URL` | `http://127.0.0.1:8080` |

Env vars take precedence over the config file.

## License

Apache 2.0
