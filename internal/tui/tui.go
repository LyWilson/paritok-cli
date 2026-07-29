package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/LyWilson/paritok-cli/internal/agent"
	"github.com/LyWilson/paritok-cli/internal/client"
)

var (
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	aiStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	codeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("118"))
)

type streamMsg struct {
	content string
	done    bool
	err     error
}

type compressResult struct {
	compressed string
	original   string
	err        error
}

type codeIterMsg struct {
	output string
	err    error
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
	content  strings.Builder

	streaming   bool
	streamChan  chan streamMsg
	respBuilder strings.Builder

	codeMode      bool
	codeIteration int

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
		streamChan: make(chan streamMsg, 64),
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
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
			m.content.WriteString(infoStyle.Render("Connected - model: " + m.modelName))
			m.viewport.SetContent(wrapContent(m.content.String(), msg.Width))
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
				m.content.WriteString("\n" + infoStyle.Render("Commands: /help, /quit, /exit, /code <prompt>") + "\n")
				m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
				m.input.SetValue("")
				m.viewport.GotoBottom()
				break
			}

			if strings.HasPrefix(input, "/code ") {
				prompt := strings.TrimPrefix(input, "/code ")
				m.codeMode = true
				m.codeIteration = 0
				m.content.WriteString("\n" + infoStyle.Render("=== CODE MODE ===") + "\n" + userStyle.Render("You: ") + prompt + "\n")
				m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
				m.viewport.GotoBottom()
				m.input.SetValue("")
				m.input.Blur()
				m.streaming = true
				m.respBuilder.Reset()
				sysPrompt := prompt + "\n\nReturn ONLY code in a single ``` block with no explanation."
				m.messages = append(m.messages, client.Message{Role: "user", Content: sysPrompt})
				m.streamChan = make(chan streamMsg, 64)
				go m.stream(m.streamChan)
				return m, waitForStream(m.streamChan)
			}

			m.origContent = input
			m.content.WriteString("\n" + userStyle.Render("You: ") + input + "\n")
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
			m.input.SetValue("")
			m.input.Blur()
			m.streaming = true
			m.respBuilder.Reset()
			m.content.WriteString(infoStyle.Render("Compressing...") + "\n")
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
			return m, compressCmd(m.paritokKey, input)
		}

	case compressResult:
		if msg.err != nil {
			m.messages = append(m.messages, client.Message{Role: "user", Content: m.origContent})
		} else {
			saved := len(msg.original) - len(msg.compressed)
			if saved > 0 {
				m.content.WriteString(infoStyle.Render(fmt.Sprintf(" (compressed: %d chars)", saved)))
				m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
				m.viewport.GotoBottom()
			}
			m.messages = append(m.messages, client.Message{Role: "user", Content: msg.compressed})
		}
		m.content.WriteString(aiStyle.Render("AI: "))
		m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
		m.viewport.GotoBottom()
		m.respBuilder.Reset()
		m.streamChan = make(chan streamMsg, 64)
		go m.stream(m.streamChan)
		return m, waitForStream(m.streamChan)

	case streamMsg:
		if msg.err != nil {
			m.content.WriteString(errStyle.Render("Error: " + msg.err.Error() + "\n"))
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
			m.streaming = false
			m.input.Focus()
			m.codeMode = false
			return m, nil
		}
		if msg.content != "" {
			m.respBuilder.WriteString(msg.content)
			m.content.WriteString(msg.content)
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
		}
		if msg.done {
			resp := m.respBuilder.String()
			m.messages = append(m.messages, client.Message{Role: "assistant", Content: resp})
			m.content.WriteString("\n")
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()

			if m.codeMode && m.codeIteration < 5 {
				m.codeIteration++
				code := agent.ExtractCodeBlock(resp)
				if code == "" {
					m.content.WriteString(errStyle.Render("No code block found, exiting code mode") + "\n")
					m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
					m.viewport.GotoBottom()
					m.streaming = false
					m.codeMode = false
					m.input.Focus()
					return m, nil
				}
				lang := agent.DetectLang(resp)
				m.content.WriteString(infoStyle.Render("Running code (" + lang + ")...") + "\n")
				m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
				m.viewport.GotoBottom()
				m.streaming = true
				return m, codeIterCmd(code, lang)
			}

			m.streaming = false
			m.codeMode = false
			m.input.Focus()
			return m, nil
		}
		return m, waitForStream(m.streamChan)

	case codeIterMsg:
		var feedback string
		if msg.err != nil {
			m.content.WriteString(errStyle.Render("Error: " + msg.err.Error() + "\n"))
			feedback = "Code error:\n" + msg.err.Error() + "\nFix the code and return ONLY the corrected version."
		} else {
			outputText := msg.output
			if outputText == "" {
				outputText = "(no output)"
			}
			m.content.WriteString(codeStyle.Render(outputText) + "\n")
			feedback = "The code ran with this output:\n" + outputText + "\nIf the result looks complete, return the FINAL code version. Otherwise improve it and return ONLY the updated code."
		}
		m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
		m.viewport.GotoBottom()

		if m.codeMode && m.codeIteration < 5 {
			m.content.WriteString(infoStyle.Render("Iterating...") + "\n")
			m.viewport.SetContent(wrapContent(m.content.String(), m.viewport.Width))
			m.viewport.GotoBottom()
			m.respBuilder.Reset()
			m.messages = append(m.messages, client.Message{Role: "user", Content: feedback})
			m.streamChan = make(chan streamMsg, 64)
			go m.stream(m.streamChan)
			return m, waitForStream(m.streamChan)
		}

		m.streaming = false
		m.codeMode = false
		m.input.Focus()
		return m, nil
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

func (m *model) stream(ch chan streamMsg) {
	ctx := context.Background()
	events, err := m.client.StreamChat(ctx, m.modelName, m.messages, nil)
	if err != nil {
		ch <- streamMsg{err: err}
		return
	}
	for evt := range events {
		switch evt.Type {
		case client.EventText:
			ch <- streamMsg{content: evt.Content}
		case client.EventDone:
			ch <- streamMsg{done: true}
			return
		case client.EventError:
			ch <- streamMsg{err: evt.Error}
			return
		}
	}
}

func codeIterCmd(code, lang string) tea.Cmd {
	return func() tea.Msg {
		output, err := agent.RunCode(code, lang)
		if err != nil {
			return codeIterMsg{err: err}
		}
		return codeIterMsg{output: output}
	}
}

func compressCmd(apiKey, content string) tea.Cmd {
	return func() tea.Msg {
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
			return compressResult{err: fmt.Errorf("compress: %w", err)}
		}
		defer resp.Body.Close()
		var r struct {
			Compressed string `json:"compressed"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return compressResult{err: fmt.Errorf("compress decode: %w", err)}
		}
		if r.Compressed == "" {
			return compressResult{compressed: content, original: content}
		}
		return compressResult{compressed: r.Compressed, original: content}
	}
}

func waitForStream(ch chan streamMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamMsg{done: true}
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
