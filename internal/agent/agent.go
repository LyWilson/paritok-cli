package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ExtractCodeBlock(resp string) string {
	start := strings.Index(resp, "```")
	if start == -1 {
		return ""
	}
	rest := resp[start+3:]
	end := strings.Index(rest, "```")
	if end == -1 {
		return ""
	}
	block := rest[:end]
	if idx := strings.IndexByte(block, '\n'); idx != -1 {
		block = block[idx+1:]
	}
	return strings.TrimSpace(block)
}

func DetectLang(resp string) string {
	start := strings.Index(resp, "```")
	if start == -1 {
		return ""
	}
	rest := resp[start+3:]
	idx := strings.IndexByte(rest, '\n')
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(rest[:idx])
}

func extForLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "typescript", "ts":
		return ".ts"
	case "javascript", "js":
		return ".js"
	case "python", "py":
		return ".py"
	case "go":
		return ".go"
	case "bash", "sh":
		return ".sh"
	default:
		return ".txt"
	}
}

func RunCode(code, lang string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "paritok-code-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	filename := "code" + extForLang(lang)
	path := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("write code: %w", err)
	}

	cmd := buildCmd(lang, path)
	if cmd == nil {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
	cmd.Dir = tmpDir

	done := make(chan struct{})
	var output []byte
	var execErr error

	go func() {
		output, execErr = cmd.CombinedOutput()
		close(done)
	}()

	select {
	case <-done:
		outStr := strings.TrimSpace(string(output))
		if execErr != nil {
			if outStr != "" {
				return outStr, nil
			}
			return "", fmt.Errorf("exec: %w", execErr)
		}
		return outStr, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return "", fmt.Errorf("execution timed out (30s)")
	}
}

func buildCmd(lang, filePath string) *exec.Cmd {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "typescript", "ts":
		return exec.Command("npx", "--yes", "tsx", filePath)
	case "javascript", "js":
		return exec.Command("node", filePath)
	case "python", "py":
		return exec.Command("python", filePath)
	case "go":
		return exec.Command("go", "run", filePath)
	case "bash", "sh":
		return exec.Command("bash", filePath)
	default:
		return nil
	}
}
