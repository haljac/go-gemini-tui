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

	"github.com/haljac/gemini-tui/internal/tools"
)

// version is set via ldflags at build time
var version = "dev"

// Available models - ordered from fastest/cheapest to most capable
const (
	ModelFlash20     = "gemini-2.0-flash"
	ModelFlash25     = "gemini-2.5-flash"
	ModelPro25       = "gemini-2.5-pro"
	ModelFlash3      = "gemini-3-flash-preview"
	ModelPro3        = "gemini-3-pro-preview"
)

var availableModels = []string{
	ModelFlash20,
	ModelFlash25,
	ModelPro25,
	ModelFlash3,
	ModelPro3,
}

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

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Italic(true)

	thinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)
)

type message struct {
	role      string
	content   string
	thinking  string   // Model's thinking process (if thinking mode enabled)
	toolsUsed []string // Track which tools were used for this response
}

type model struct {
	client       *genai.Client
	toolExecutor *tools.Executor
	viewport     viewport.Model
	textarea     textarea.Model
	messages     []message
	conversation []*genai.Content // Full conversation history for API
	mdRenderer   *glamour.TermRenderer
	err          error
	ready        bool
	waiting      bool
	activeTools  []string // Tools currently being executed
	width        int
	height       int
	// Streaming state
	streaming       bool
	streamBuffer    string
	streamThinking  string
	streamToolsUsed []string
	streamChan      chan streamEvent
	// Thinking mode
	thinkingEnabled bool
	currentModel    string
	showThinking    bool // Toggle to show/hide thinking in UI
}

// Streaming event types
type streamEvent struct {
	chunk         string
	thinking      string
	done          bool
	err           error
	functionCalls []*genai.FunctionCall
	conversation  []*genai.Content
}

type responseMsg struct {
	content   string
	toolsUsed []string
	err       error
}

type functionCallMsg struct {
	calls        []*genai.FunctionCall
	conversation []*genai.Content
}

// Streaming message types
type streamChunkMsg struct {
	chunk string
}

type streamDoneMsg struct {
	fullContent string
	thinking    string
	toolsUsed   []string
}

type streamErrorMsg struct {
	err error
}

type streamFunctionCallMsg struct {
	calls        []*genai.FunctionCall
	conversation []*genai.Content
}

func initialModel(client *genai.Client, executor *tools.Executor) model {
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
		client:          client,
		toolExecutor:    executor,
		textarea:        ta,
		messages:        []message{},
		conversation:    []*genai.Content{},
		mdRenderer:      mdRenderer,
		currentModel:    ModelFlash20,
		thinkingEnabled: false,
		showThinking:    true,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) nextModel() string {
	for i, model := range availableModels {
		if model == m.currentModel {
			return availableModels[(i+1)%len(availableModels)]
		}
	}
	return availableModels[0]
}

func (m *model) sendMessage(userMsg string) tea.Cmd {
	// Build conversation with current user message
	conversation := append(m.conversation, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: userMsg}},
	})

	return m.startStreaming(conversation, nil)
}

func (m *model) continueWithFunctionResults(conversation []*genai.Content, toolsUsed []string) tea.Cmd {
	return m.startStreaming(conversation, toolsUsed)
}

func (m *model) startStreaming(conversation []*genai.Content, toolsUsed []string) tea.Cmd {
	// Create channel for streaming events
	ch := make(chan streamEvent, 10)
	m.streamChan = ch
	m.streamToolsUsed = toolsUsed

	// Start streaming in background
	go m.streamInBackground(conversation, toolsUsed, ch)

	// Return command to wait for first event
	return m.waitForStreamEvent()
}

func (m *model) streamInBackground(conversation []*genai.Content, toolsUsed []string, ch chan streamEvent) {
	defer close(ch)

	ctx := context.Background()

	// Configure with tools and system instruction
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{
				Text: `You are an expert coding agent. You help users write, modify, debug, and understand code. You can read, create, and edit files in the user's project.

## Core Principles

1. **Understand before acting**: Read relevant files before making changes. Explore the codebase to understand patterns and conventions.
2. **Make surgical edits**: Use edit_file for small changes to existing files. Use write_file for new files or complete rewrites.
3. **Explain your changes**: Briefly describe what you're doing and why.
4. **Follow existing patterns**: Match the code style, naming conventions, and architecture of the project.

## Tools Available

Reading:
- read_file: Read file contents
- list_directory: List directory contents
- glob_search: Find files by pattern (e.g., '**/*.go')

Writing:
- write_file: Create new files or overwrite existing files
- edit_file: Make surgical edits by replacing specific strings (old_string must be unique)
- create_directory: Create directories

## Best Practices

- Always read a file before editing it
- When editing, include enough context in old_string to make it unique
- Create parent directories before writing files to new paths
- For multi-file changes, handle them one at a time
- If an edit fails because old_string isn't unique, include more surrounding context`,
			}},
		},
		Tools: []*genai.Tool{{
			FunctionDeclarations: tools.AllTools(),
		}},
	}

	// Add thinking config if enabled
	if m.thinkingEnabled {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
		}
	}

	var fullText strings.Builder
	var thinkingText strings.Builder
	var functionCalls []*genai.FunctionCall
	var functionCallParts []*genai.Part // Preserve original parts with ThoughtSignature

	// Stream the response
	for resp, err := range m.client.Models.GenerateContentStream(ctx, m.currentModel, conversation, config) {
		if err != nil {
			ch <- streamEvent{err: err}
			return
		}

		// Check for function calls in this chunk
		if calls := resp.FunctionCalls(); len(calls) > 0 {
			functionCalls = append(functionCalls, calls...)
		}

		// Extract thinking and text content from response
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Thought {
					// This is thinking content
					if part.Text != "" {
						thinkingText.WriteString(part.Text)
					}
				} else if part.Text != "" {
					// Regular text content
					fullText.WriteString(part.Text)
					ch <- streamEvent{chunk: part.Text}
				} else if part.FunctionCall != nil {
					// Preserve original function call parts (includes ThoughtSignature)
					functionCallParts = append(functionCallParts, part)
				}
			}
		}
	}

	// If we have function calls, send them
	if len(functionCalls) > 0 {
		// Build the model's response content for conversation history
		// Use original parts to preserve ThoughtSignature
		var parts []*genai.Part
		if fullText.Len() > 0 {
			parts = append(parts, &genai.Part{Text: fullText.String()})
		}
		// Use the preserved original parts that include ThoughtSignature
		parts = append(parts, functionCallParts...)
		newConversation := append(conversation, &genai.Content{
			Role:  "model",
			Parts: parts,
		})
		ch <- streamEvent{
			done:          true,
			functionCalls: functionCalls,
			conversation:  newConversation,
		}
		return
	}

	// Done with text response
	ch <- streamEvent{done: true, thinking: thinkingText.String()}
}

func (m *model) waitForStreamEvent() tea.Cmd {
	return func() tea.Msg {
		if m.streamChan == nil {
			return streamErrorMsg{err: fmt.Errorf("no stream channel")}
		}

		event, ok := <-m.streamChan
		if !ok {
			// Channel closed unexpectedly
			return streamDoneMsg{fullContent: m.streamBuffer, toolsUsed: m.streamToolsUsed}
		}

		if event.err != nil {
			return streamErrorMsg{err: event.err}
		}

		if event.done {
			if len(event.functionCalls) > 0 {
				return streamFunctionCallMsg{
					calls:        event.functionCalls,
					conversation: event.conversation,
				}
			}
			return streamDoneMsg{
				fullContent: m.streamBuffer,
				thinking:    event.thinking,
				toolsUsed:   m.streamToolsUsed,
			}
		}

		return streamChunkMsg{chunk: event.chunk}
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
			if m.waiting || m.streaming {
				return m, nil
			}
			userInput := strings.TrimSpace(m.textarea.Value())
			if userInput == "" {
				return m, nil
			}
			m.messages = append(m.messages, message{role: "user", content: userInput})
			m.textarea.Reset()
			m.waiting = true
			m.streaming = true
			m.streamBuffer = ""
			m.streamThinking = ""
			m.activeTools = nil
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			cmd := m.sendMessage(userInput)
			return m, cmd
		}
		// Handle other key combinations
		switch msg.String() {
		case "ctrl+t":
			// Toggle thinking mode
			m.thinkingEnabled = !m.thinkingEnabled
			m.viewport.SetContent(m.renderMessages())
			return m, nil
		case "ctrl+g":
			// Cycle through models
			m.currentModel = m.nextModel()
			m.viewport.SetContent(m.renderMessages())
			return m, nil
		case "ctrl+h":
			// Toggle showing thinking in UI
			m.showThinking = !m.showThinking
			m.viewport.SetContent(m.renderMessages())
			return m, nil
		}

	case streamChunkMsg:
		// Append chunk to buffer and update display
		m.streamBuffer += msg.chunk
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		cmd := m.waitForStreamEvent()
		return m, cmd

	case streamDoneMsg:
		// Streaming complete - finalize the message
		m.waiting = false
		m.streaming = false
		m.activeTools = nil
		content := msg.fullContent
		if content == "" {
			content = m.streamBuffer
		}
		m.messages = append(m.messages, message{
			role:      "assistant",
			content:   content,
			thinking:  msg.thinking,
			toolsUsed: msg.toolsUsed,
		})
		m.streamBuffer = ""
		m.streamThinking = ""
		// Update conversation history for next turn
		m.conversation = append(m.conversation,
			&genai.Content{
				Role:  "user",
				Parts: []*genai.Part{{Text: m.messages[len(m.messages)-2].content}},
			},
			&genai.Content{
				Role:  "model",
				Parts: []*genai.Part{{Text: content}},
			},
		)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case streamErrorMsg:
		m.waiting = false
		m.streaming = false
		m.streamBuffer = ""
		m.err = msg.err
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case streamFunctionCallMsg:
		// Execute the function calls
		m.streaming = false
		m.streamBuffer = ""
		var toolNames []string
		var functionResponses []*genai.Part

		for _, call := range msg.calls {
			toolNames = append(toolNames, call.Name)
			result, _ := m.toolExecutor.Execute(call.Name, call.Args)
			functionResponses = append(functionResponses, genai.NewPartFromFunctionResponse(call.Name, result))
		}

		// Update active tools for UI feedback
		m.activeTools = toolNames
		m.streamToolsUsed = toolNames
		m.streaming = true
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

		// Add function responses to conversation
		conversation := append(msg.conversation, &genai.Content{
			Role:  "user",
			Parts: functionResponses,
		})

		// Continue the conversation with function results
		cmd := m.continueWithFunctionResults(conversation, toolNames)
		return m, cmd

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
		return infoStyle.Render("Start a conversation with Gemini. Type your message and press Enter.\nGemini can read files - try asking about files in your project!")
	}

	var sb strings.Builder
	for _, msg := range m.messages {
		if msg.role == "user" {
			sb.WriteString(userStyle.Render("You: "))
			sb.WriteString(msg.content)
			sb.WriteString("\n\n")
		} else {
			// Show thinking if present and enabled
			if msg.thinking != "" && m.showThinking {
				sb.WriteString(thinkingStyle.Render("Thinking:"))
				sb.WriteString("\n")
				sb.WriteString(thinkingStyle.Render(msg.thinking))
				sb.WriteString("\n\n")
			}
			// Show tools used if any
			if len(msg.toolsUsed) > 0 {
				sb.WriteString(toolStyle.Render("Tools used: "))
				sb.WriteString(toolStyle.Render(strings.Join(msg.toolsUsed, ", ")))
				sb.WriteString("\n")
			}
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

	// Show streaming content
	if m.streaming && m.streamBuffer != "" {
		if len(m.streamToolsUsed) > 0 {
			sb.WriteString(toolStyle.Render("Tools used: "))
			sb.WriteString(toolStyle.Render(strings.Join(m.streamToolsUsed, ", ")))
			sb.WriteString("\n")
		}
		sb.WriteString(assistantStyle.Render("Gemini:"))
		sb.WriteString("\n")
		// Show raw text while streaming (markdown rendering can be janky mid-stream)
		sb.WriteString(m.streamBuffer)
		sb.WriteString(infoStyle.Render("..."))
		sb.WriteString("\n\n")
	} else if m.waiting {
		if len(m.activeTools) > 0 {
			sb.WriteString(toolStyle.Render("Using tools: "))
			sb.WriteString(toolStyle.Render(strings.Join(m.activeTools, ", ")))
			sb.WriteString("\n")
		}
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

	// Build status bar
	modelStatus := statusStyle.Render(m.currentModel)
	thinkingStatus := statusStyle.Render("Thinking: OFF")
	if m.thinkingEnabled {
		thinkingStatus = statusActiveStyle.Render("Thinking: ON")
	}
	statusBar := fmt.Sprintf("%s %s", modelStatus, thinkingStatus)

	header := titleStyle.Render("Gemini TUI") + "  " + statusBar
	footer := m.textarea.View()
	help := infoStyle.Render("Enter: send | Ctrl+T: thinking | Ctrl+G: model | Ctrl+H: hide thinking | Esc: quit")

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, m.viewport.View(), footer, help)
}

func main() {
	// Handle --version flag
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("gemini-tui %s\n", version)
			os.Exit(0)
		case "--help", "-h", "help":
			fmt.Println("gemini-tui - A terminal UI for Google Gemini")
			fmt.Printf("Version: %s\n\n", version)
			fmt.Println("Usage: gemini-tui [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --version, -v    Show version")
			fmt.Println("  --help, -h       Show this help")
			fmt.Println()
			fmt.Println("Environment:")
			fmt.Println("  GOOGLE_API_KEY   Required. Your Gemini API key")
			fmt.Println()
			fmt.Println("Available models (cycle with Ctrl+G):")
			for _, m := range availableModels {
				fmt.Printf("  - %s\n", m)
			}
			fmt.Println()
			fmt.Println("Keyboard shortcuts:")
			fmt.Println("  Enter      Send message")
			fmt.Println("  Ctrl+T     Toggle thinking mode")
			fmt.Println("  Ctrl+G     Cycle models")
			fmt.Println("  Ctrl+H     Toggle thinking display")
			fmt.Println("  Esc        Quit")
			fmt.Println()
			fmt.Println("Get an API key at: https://aistudio.google.com/apikey")
			os.Exit(0)
		}
	}

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

	// Create tool executor rooted at current working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	executor, err := tools.NewExecutor(wd)
	if err != nil {
		fmt.Printf("Error creating tool executor: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		initialModel(client, executor),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
