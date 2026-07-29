package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yourusername/paritok-cli/internal/client"
)

func Definitions() []client.ToolDefinition {
	return []client.ToolDefinition{
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "read",
				Description: "Read the contents of a file. Handles text files, images, and PDFs.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Absolute path or path relative to the project root"}},"required":["path"]}`),
			},
		},
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "write",
				Description: "Write content to a file. Creates the file if it doesn't exist.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"},"content":{"type":"string","description":"Content to write"}},"required":["path","content"]}`),
			},
		},
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "edit",
				Description: "Edit a file by replacing text. Finds the first occurrence of oldString and replaces it with newString.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"},"oldString":{"type":"string","description":"Text to find and replace"},"newString":{"type":"string","description":"Replacement text"}},"required":["path","oldString","newString"]}`),
			},
		},
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "glob",
				Description: "Find files matching a glob pattern. Examples: **/*.go, src/**/*.tsx",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string","description":"Glob pattern to match"}},"required":["pattern"]}`),
			},
		},
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "grep",
				Description: "Search file contents using a regex pattern. Returns matching lines with file paths and line numbers.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string","description":"Regex pattern to search"},"include":{"type":"string","description":"Optional file glob filter (e.g. *.go)"}},"required":["pattern"]}`),
			},
		},
		{
			Type: "function",
			Function: client.FunctionDef{
				Name:        "bash",
				Description: "Execute a shell command and return the output. Use for running scripts, building, testing, or any terminal operation.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"Shell command to execute"}},"required":["command"]}`),
			},
		},
	}
}

type readParams struct {
	Path string `json:"path"`
}

type writeParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type editParams struct {
	Path      string `json:"path"`
	OldString string `json:"oldString"`
	NewString string `json:"newString"`
}

type globParams struct {
	Pattern string `json:"pattern"`
}

type grepParams struct {
	Pattern string `json:"pattern"`
	Include string `json:"include,omitempty"`
}

type bashParams struct {
	Command string `json:"command"`
}

func Dispatch(name string, args json.RawMessage) (string, error) {
	switch name {
	case "read":
		var p readParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("read: %w", err)
		}
		return handleRead(p)
	case "write":
		var p writeParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("write: %w", err)
		}
		return handleWrite(p)
	case "edit":
		var p editParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("edit: %w", err)
		}
		return handleEdit(p)
	case "glob":
		var p globParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("glob: %w", err)
		}
		return handleGlob(p)
	case "grep":
		var p grepParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("grep: %w", err)
		}
		return handleGrep(p)
	case "bash":
		var p bashParams
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("bash: %w", err)
		}
		return handleBash(p)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func handleRead(p readParams) (string, error) {
	data, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	return string(data), nil
}

func handleWrite(p writeParams) (string, error) {
	dir := filepath.Dir(p.Path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("write %s: %w", p.Path, err)
		}
	}
	if err := os.WriteFile(p.Path, []byte(p.Content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	return "Written " + p.Path, nil
}

func handleEdit(p editParams) (string, error) {
	data, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("edit %s: %w", p.Path, err)
	}
	content := string(data)
	idx := strings.Index(content, p.OldString)
	if idx == -1 {
		return "", fmt.Errorf("edit %s: oldString not found", p.Path)
	}
	content = content[:idx] + p.NewString + content[idx+len(p.OldString):]
	if err := os.WriteFile(p.Path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("edit %s: %w", p.Path, err)
	}
	return "Edited " + p.Path, nil
}

func handleGlob(p globParams) (string, error) {
	matches, err := filepath.Glob(p.Pattern)
	if err != nil {
		return "", fmt.Errorf("glob: %w", err)
	}
	if len(matches) == 0 {
		return "No files matched", nil
	}
	return strings.Join(matches, "\n"), nil
}

func handleGrep(p grepParams) (string, error) {
	cmd := exec.Command("rg", "-n", p.Pattern)
	if p.Include != "" {
		cmd.Args = append(cmd.Args, "-g", p.Include)
	}
	cmd.Args = append(cmd.Args, ".")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("grep: %s", string(exitErr.Stderr))
		}
	}
	if len(out) == 0 {
		return "No matches found", nil
	}
	return string(out), nil
}

func handleBash(p bashParams) (string, error) {
	cmd := exec.Command("powershell", "-Command", p.Command)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		if result != "" {
			return result, nil
		}
		return "", fmt.Errorf("bash: %w", err)
	}
	return result, nil
}
