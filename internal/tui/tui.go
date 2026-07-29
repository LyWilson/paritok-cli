package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/yourusername/paritok-cli/internal/client"
	"github.com/yourusername/paritok-cli/internal/tools"
)

	var (
	userStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	aiStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	toolCallStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	toolResStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("118"))
	logoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	subStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cmdStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

type chatEntry struct {
	kind    string // "user", "assistant", "tool_call", "tool_result", "system"
	content string // raw text
}

type compressResult struct {
	compressed string
	original   string
	err        error
}

type model struct {
	client      *client.Client
	paritokKey  string
	modelName   string
	messages    []client.Message
	origContent string

	viewport viewport.Model
	input    textinput.Model
	ready    bool
	entries  []chatEntry
	content  strings.Builder

	streaming   bool
	agentChan   chan agentMsg
	currentResp strings.Builder
	welcomeShown bool

	err error
}

func New(cl *client.Client, paritokKey, modelName string) *model {
	ti := textinput.New()
	ti.Placeholder = "Message (/help for commands)"
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 80

	return &model{
		client:     cl,
		paritokKey: paritokKey,
		modelName:  modelName,
		input:      ti,
		agentChan:  make(chan agentMsg, 256),
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) appendEntry(e chatEntry) {
	m.entries = append(m.entries, e)
	switch e.kind {
	case "user":
		m.content.WriteString("\n" + userStyle.Render("You: ") + e.content + "\n")
	case "assistant":
		rendered, err := glamour.Render(e.content, "dark")
		if err != nil {
			m.content.WriteString("\n" + aiStyle.Render("AI: ") + e.content + "\n")
		} else {
			lines := strings.Split(rendered, "\n")
			for i, l := range lines {
				lines[i] = strings.TrimRight(l, " \t\r")
			}
			m.content.WriteString("\n" + aiStyle.Render("AI: ") + "\n" + strings.Join(lines, "\n") + "\n")
		}
	case "tool_call":
		m.content.WriteString("\n" + toolCallStyle.Render("🔧 " + e.content) + "\n")
	case "tool_result":
		m.content.WriteString(toolResStyle.Render(e.content) + "\n")
	case "system":
		m.content.WriteString("\n" + infoStyle.Render(e.content) + "\n")
	}
	m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
	m.viewport.GotoBottom()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 3
		if h < 1 {
			h = 1
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, h)
			m.viewport.YPosition = 0
			m.input.Width = msg.Width
			m.ready = true
			if !m.welcomeShown {
				m.welcomeShown = true
				m.showWelcome()
			}
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = h
			m.input.Width = msg.Width
			m.viewport.SetContent(wrapContent(m.content.String(), msg.Width))
		}
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "pgup", "pgdown", "up", "down", "home", "end":
			if !m.ready {
				break
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case "ctrl+c":
			return m, tea.Quit
		}

		if m.streaming {
			return m, nil
		}

		switch msg.String() {
		case "enter":
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				break
			}

			if input == "/quit" || input == "/exit" {
				return m, tea.Quit
			}
			if input == "/help" {
				m.appendEntry(chatEntry{kind: "system", content: "Commands: /help, /quit, /exit"})
				m.input.SetValue("")
				break
			}

			m.origContent = input
			m.appendEntry(chatEntry{kind: "user", content: input})
			m.input.SetValue("")
			m.input.Blur()
			m.streaming = true
			m.appendEntry(chatEntry{kind: "system", content: "Compressing..."})
			return m, compressCmd(m.paritokKey, input)
		}

	case compressResult:
		if msg.err != nil {
			m.messages = append(m.messages, client.Message{Role: "user", Content: m.origContent})
		} else {
			saved := len(msg.original) - len(msg.compressed)
			if saved > 0 {
				m.appendEntry(chatEntry{kind: "system", content: fmt.Sprintf("Compressed: %d chars saved", saved)})
			}
			m.messages = append(m.messages, client.Message{Role: "user", Content: msg.compressed})
		}
		go runAgent(m.client, m.modelName, m.messages, tools.Definitions(), m.agentChan)
		return m, waitForAgent(m.agentChan)

	case agentMsg:
		switch msg.kind {
		case "text":
			m.currentResp.WriteString(msg.content)
			return m, waitForAgent(m.agentChan)

		case "tool_call":
			if m.currentResp.Len() > 0 {
				m.appendEntry(chatEntry{kind: "assistant", content: m.currentResp.String()})
				m.currentResp.Reset()
			}
			m.appendEntry(chatEntry{kind: "tool_call", content: msg.toolName + "(" + msg.content + ")"})

		case "tool_result":
			m.appendEntry(chatEntry{kind: "tool_result", content: msg.content})

		case "done":
			if m.currentResp.Len() > 0 {
				m.appendEntry(chatEntry{kind: "assistant", content: m.currentResp.String()})
				m.currentResp.Reset()
			}
			m.content.WriteString("\n")
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
			m.streaming = false
			m.input.Focus()
			if msg.messages != nil {
				m.messages = msg.messages
			}

		case "error":
			if m.currentResp.Len() > 0 {
				m.appendEntry(chatEntry{kind: "assistant", content: m.currentResp.String()})
				m.currentResp.Reset()
			}
			m.appendEntry(chatEntry{kind: "system", content: "Error: " + msg.content})
			m.streaming = false
			m.input.Focus()
		}
		return m, waitForAgent(m.agentChan)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return m.viewport.View() + "\n" + m.input.View()
}

func (m *model) showWelcome() {
	logo := logoStyle.Render(`██████╗  █████╗ ██████╗ ██╗████████╗ ██████╗ ██╗  ██╗
██╔══██╗██╔══██╗██╔══██╗██║╚══██╔══╝██╔═══██╗██║ ██╔╝
██████╔╝███████║██████╔╝██║   ██║   ██║   ██║█████╔╝ 
██╔═══╝ ██╔══██║██╔══██╗██║   ██║   ██║   ██║██╔═██╗ 
██║     ██║  ██║██║  ██║██║   ██║   ╚██████╔╝██║  ██╗
╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝`)
	m.content.WriteString("\n" + logo + "\n\n")
	m.content.WriteString(subStyle.Render("  Cost-optimized coding agent CLI") + "\n")
	m.content.WriteString(infoStyle.Render("  Model: "+m.modelName) + "\n\n")
	m.content.WriteString(cmdStyle.Render("  /help") + infoStyle.Render(" — show commands") + "\n")
	m.content.WriteString(cmdStyle.Render("  /quit") + infoStyle.Render(" — exit") + "\n\n")
	m.content.WriteString(infoStyle.Render("  Type a message or ask me to build something...") + "\n")
	m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
	m.viewport.GotoBottom()
}

func compressCmd(apiKey, content string) tea.Cmd {
	return func() tea.Msg {
		if apiKey == "" {
			return compressResult{compressed: content, original: content}
		}
		body, _ := json.Marshal(map[string]string{
			"content": content,
			"query":   content,
			"kind":    "prompt",
		})
		req, _ := http.NewRequest("POST", "https://www.paritok.com/api/compress", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return compressResult{compressed: content, original: content}
		}
		defer resp.Body.Close()
		var r struct {
			Compressed string `json:"compressed"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return compressResult{compressed: content, original: content}
		}
		if r.Compressed == "" {
			return compressResult{compressed: content, original: content}
		}
		return compressResult{compressed: r.Compressed, original: content}
	}
}

func waitForAgent(ch chan agentMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return agentMsg{kind: "done"}
		}
		return msg
	}
}

func wrapContent(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		w := ansi.StringWidth(line)
		if w <= width {
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteByte('\n')
			continue
		}
		current := ""
		for _, word := range words {
			candidate := word
			if current != "" {
				candidate = current + " " + word
			}
			if ansi.StringWidth(candidate) > width {
				if current != "" {
					result.WriteString(current)
					result.WriteByte('\n')
				}
				current = word
			} else {
				current = candidate
			}
		}
		if current != "" {
			result.WriteString(current)
		}
		result.WriteByte('\n')
	}
	return result.String()
}
