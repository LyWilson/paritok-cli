package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type ServerConfig struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type Tool struct {
	ServerName string
	Tool       mcp.Tool
}

type Server struct {
	config ServerConfig
	name   string
	cli    *client.StdioMCPClient
	mu     sync.Mutex
}

type Manager struct {
	servers map[string]*Server
	tools   []Tool
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{servers: make(map[string]*Server)}
}

func LoadConfig(path string) (map[string]ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mcp config: %w", err)
	}

	var wrapper struct {
		MCPServers map[string]ServerConfig `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}

	return wrapper.MCPServers, nil
}

func (m *Manager) AddServer(name string, cfg ServerConfig) error {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if len(cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), envSlice(cfg.Env)...)
	}

	cli, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	if err != nil {
		return fmt.Errorf("create mcp client for %q: %w", name, err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "paritok-cli",
		Version: "0.1.0",
	}

	if _, err := cli.Initialize(ctx, initReq); err != nil {
		cli.Close()
		return fmt.Errorf("initialize mcp server %q: %w", name, err)
	}

	listReq := mcp.ListToolsRequest{}
	toolsResult, err := cli.ListTools(ctx, listReq)
	if err != nil {
		cli.Close()
		return fmt.Errorf("list tools from %q: %w", name, err)
	}

	svr := &Server{
		config: cfg,
		name:   name,
		cli:    cli,
	}

	m.mu.Lock()
	m.servers[name] = svr
	for _, t := range toolsResult.Tools {
		m.tools = append(m.tools, Tool{
			ServerName: name,
			Tool:       t,
		})
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) Tools() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Tool, len(m.tools))
	copy(result, m.tools)
	return result
}

func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
	m.mu.RLock()
	svr, ok := m.servers[serverName]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("mcp server %q not found", serverName)
	}

	svr.mu.Lock()
	defer svr.mu.Unlock()

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = toolName
	callReq.Params.Arguments = args

	result, err := svr.cli.CallTool(ctx, callReq)
	if err != nil {
		return "", fmt.Errorf("call tool %q on %q: %w", toolName, serverName, err)
	}

	var parts []string
	for _, content := range result.Content {
		switch v := content.(type) {
		case mcp.TextContent:
			parts = append(parts, v.Text)
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}

	return strings.Join(parts, "\n"), nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, svr := range m.servers {
		svr.cli.Close()
	}
}

func envSlice(env map[string]string) []string {
	var s []string
	for k, v := range env {
		s = append(s, k+"="+v)
	}
	return s
}
