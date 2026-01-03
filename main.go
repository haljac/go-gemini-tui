package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/genai"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
)

type message struct {
	role    string
	content string
}

type model struct {
	client     *genai.Client
	viewport   viewport.Model
	textarea   textarea.Model
	messages   []message
	mdRenderer *glamour.TermRenderer
	err        error
	ready      bool
	waiting    bool
	width      int
	height     int
}

type responseMsg struct {
	content string
	err     error
}

func initialModel(client *genai.Client) model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Create markdown renderer (defer to window size handler for proper width)
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(80),
	)

	return model{
		client:     client,
		textarea:   ta,
		messages:   []message{},
		mdRenderer: mdRenderer,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) sendMessage(userMsg string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Build conversation history
		var contents []*genai.Content
		for _, msg := range m.messages {
			role := msg.role
			if role == "user" {
				role = "user"
			} else {
				role = "model"
			}
			contents = append(contents, &genai.Content{
				Role:  role,
				Parts: []*genai.Part{{Text: msg.content}},
			})
		}

		// Add current user message
		contents = append(contents, &genai.Content{
			Role:  "user",
			Parts: []*genai.Part{{Text: userMsg}},
		})

		result, err := m.client.Models.GenerateContent(
			ctx,
			"gemini-2.0-flash",
			contents,
			nil,
		)
		if err != nil {
			return responseMsg{err: err}
		}

		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return responseMsg{err: fmt.Errorf("no response from model")}
		}

		return responseMsg{content: result.Candidates[0].Content.Parts[0].Text}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.waiting {
				return m, nil
			}
			userInput := strings.TrimSpace(m.textarea.Value())
			if userInput == "" {
				return m, nil
			}
			m.messages = append(m.messages, message{role: "user", content: userInput})
			m.textarea.Reset()
			m.waiting = true
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, m.sendMessage(userInput)
		}

	case responseMsg:
		m.waiting = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.messages = append(m.messages, message{role: "assistant", content: msg.content})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 2
		footerHeight := 5

		// Update markdown renderer with new width
		m.mdRenderer, _ = glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(msg.Width-4),
		)

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
			m.viewport.SetContent(m.renderMessages())
		}

		m.textarea.SetWidth(msg.Width - 2)
	}

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(taCmd, vpCmd)
}

func (m model) renderMessages() string {
	if len(m.messages) == 0 {
		return infoStyle.Render("Start a conversation with Gemini. Type your message and press Enter.")
	}

	var sb strings.Builder
	for _, msg := range m.messages {
		if msg.role == "user" {
			sb.WriteString(userStyle.Render("You: "))
			sb.WriteString(msg.content)
			sb.WriteString("\n\n")
		} else {
			sb.WriteString(assistantStyle.Render("Gemini:"))
			sb.WriteString("\n")
			// Render markdown for assistant messages
			if m.mdRenderer != nil {
				rendered, err := m.mdRenderer.Render(msg.content)
				if err == nil {
					sb.WriteString(strings.TrimSpace(rendered))
				} else {
					sb.WriteString(msg.content)
				}
			} else {
				sb.WriteString(msg.content)
			}
			sb.WriteString("\n\n")
		}
	}

	if m.waiting {
		sb.WriteString(infoStyle.Render("Gemini is thinking..."))
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return sb.String()
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	header := titleStyle.Render("Gemini TUI")
	footer := m.textarea.View()
	help := infoStyle.Render("Enter: send | Esc/Ctrl+C: quit")

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, m.viewport.View(), footer, help)
}

func main() {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GOOGLE_API_KEY environment variable is not set")
		fmt.Println("Get your API key from: https://aistudio.google.com/apikey")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Printf("Error creating Gemini client: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		initialModel(client),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
