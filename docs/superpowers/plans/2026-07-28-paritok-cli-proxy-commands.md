# Proxy Management Commands Implementation Plan

**Goal:** Add `paritok-cli start`, `paritok-cli stop`, and `paritok-cli auth-nvidia` commands

**Architecture:** Extend `~/.paritok.json` with `nvidia_api_key`. New `start` command finds `paritok.exe`, launches proxy with correct env vars. `stop` kills it.

**Tech Stack:** Go, cobra, os/exec

---

### Task 1: Extend config with NvidiaAPIKey

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Add NvidiaAPIKey field and update Save**

```go
type Config struct {
	APIKey       string `json:"api_key"`
	NvidiaAPIKey string `json:"nvidia_api_key,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

func Save(apiKey string) error {
	cfg := Config{APIKey: apiKey}
	data, err := json.MarshalIndent(cfg, "", "  ")
	// ... same as before
}

func SaveNvidia(apiKey string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	cfg := Config{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &cfg)
	}
	cfg.NvidiaAPIKey = apiKey
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
```

- [ ] **Build and verify**

Run: `cd C:\Users\wilso\Documents\paritok-cli && go build ./internal/config/...`

### Task 2: Add auth-nvidia command

**Files:**
- Modify: `cmd/root.go`

- [ ] **Add authNvidiaCmd**

```go
var authNvidiaCmd = &cobra.Command{
	Use:   "auth-nvidia <api_key>",
	Short: "Save your NVIDIA API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if key == "" {
			return fmt.Errorf("api key cannot be empty")
		}
		if err := config.SaveNvidia(key); err != nil {
			return fmt.Errorf("save nvidia api key: %w", err)
		}
		fmt.Fprintln(os.Stderr, "NVIDIA API key saved")
		return nil
	},
}

func Execute() error {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(authNvidiaCmd)
	return rootCmd.Execute()
}
```

### Task 3: Create proxy start/stop commands

**Files:**
- Create: `cmd/proxy.go`

- [ ] **Create cmd/proxy.go**

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/paritok-cli/internal/config"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage the paritok proxy",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the paritok proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.NvidiaAPIKey == "" {
			return fmt.Errorf("NVIDIA API key not set — run 'paritok-cli auth-nvidia <key>'")
		}

		paritokPath, err := findParitok()
		if err != nil {
			return err
		}

		workDir := getWorkDir()

		proc := exec.Command(paritokPath,
			"proxy", "--port", "8080",
			"--config-file", filepath.Join(workDir, "paritok.yaml"),
			"--openai-url", "https://integrate.api.nvidia.com",
		)
		proc.Dir = workDir
		proc.Env = append(os.Environ(),
			"PARITOK_API_KEY="+cfg.APIKey,
			"NVIDIA_API_KEY="+cfg.NvidiaAPIKey,
		)

		if err := proc.Start(); err != nil {
			return fmt.Errorf("start proxy: %w", err)
		}

		// Wait for health endpoint
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			if err := checkHealth(); err == nil {
				fmt.Fprintf(os.Stderr, "Proxy running on http://127.0.0.1:8080 (pid %d)\n", proc.Process.Pid)
				return nil
			}
		}
		return fmt.Errorf("proxy started but health check timed out")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the paritok proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := exec.Command("taskkill", "/IM", "paritok.exe", "/F").Run(); err != nil {
			return fmt.Errorf("stop proxy: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Proxy stopped")
		return nil
	},
}

func findParitok() (string, error) {
	if p, err := exec.LookPath("paritok"); err == nil {
		return p, nil
	}
	// Check common Python Scripts locations
	dirs := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Python", "pythoncore-3.14-64", "Scripts"),
		filepath.Join(os.Getenv("APPDATA"), "Python", "Scripts"),
	}
	for _, d := range dirs {
		p := filepath.Join(d, "paritok.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("paritok not found on PATH or in common Python Scripts directories")
}

func getWorkDir() string {
	if d := os.Getenv("PARITOK_CLI_DIR"); d != "" {
		return d
	}
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func checkHealth() error {
	// Simple TCP dial to check if proxy is listening
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8080", 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func init() {
	proxyCmd.AddCommand(startCmd)
	proxyCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(proxyCmd)
}
```

- [ ] **Build and verify**

Run: `go build -o paritok-cli.exe .` — should succeed with no errors.

### Task 4: End-to-end test

- [ ] **Test auth-nvidia**

Run: `.\paritok-cli auth-nvidia nvapi-test-key` — should print "NVIDIA API key saved"

- [ ] **Test start**

Run: `.\paritok-cli start` — should find paritok, launch it, wait for health check

- [ ] **Test stop**

Run: `.\paritok-cli stop` — should kill paritok.exe

- [ ] **Test full flow**

```powershell
paritok-cli auth-nvidia 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
paritok-cli start
paritok-cli chat
```