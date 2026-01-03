# Feature Implementation Plan

This document outlines the implementation plan for the three highest-value features for the Gemini TUI.

## Feature Overview

| Feature | Priority | Complexity | Value |
|---------|----------|------------|-------|
| 1. File System Tool Use | High | Medium | Enables code analysis, file reading, project understanding |
| 2. Streaming Responses | High | Medium | Dramatically improves perceived responsiveness |
| 3. Thinking Mode | Medium | Low | Better reasoning for complex coding/math tasks |

---

## Feature 1: File System Tool Use

### Overview

Enable Gemini to read files from the local filesystem through function calling. This transforms the TUI from a simple chat interface into a powerful developer assistant that can analyze code, read documentation, and understand project context.

### Capabilities

- **Read File**: Read contents of a single file by path
- **List Directory**: List files and directories at a path
- **Glob Search**: Find files matching a pattern (e.g., `**/*.go`)

### Implementation Details

#### 1.1 Define Tool Declarations

Create function declarations for each tool:

```go
// internal/tools/tools.go
package tools

import "google.golang.org/genai"

var ReadFileTool = &genai.FunctionDeclaration{
    Name:        "read_file",
    Description: "Read the contents of a file at the given path. Use this to examine source code, configuration files, or any text file.",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "path": {
                Type:        genai.TypeString,
                Description: "The absolute or relative file path to read",
            },
        },
        Required: []string{"path"},
    },
}

var ListDirectoryTool = &genai.FunctionDeclaration{
    Name:        "list_directory",
    Description: "List files and directories at the given path. Returns names with type indicators (/ for directories).",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "path": {
                Type:        genai.TypeString,
                Description: "The directory path to list. Defaults to current directory if empty.",
            },
        },
        Required: []string{},
    },
}

var GlobSearchTool = &genai.FunctionDeclaration{
    Name:        "glob_search",
    Description: "Find files matching a glob pattern. Useful for finding all files of a type (e.g., '**/*.go' for all Go files).",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "pattern": {
                Type:        genai.TypeString,
                Description: "Glob pattern to match (e.g., '*.go', 'src/**/*.ts')",
            },
        },
        Required: []string{"pattern"},
    },
}
```

#### 1.2 Implement Tool Executors

```go
// internal/tools/executor.go
package tools

import (
    "fmt"
    "os"
    "path/filepath"
)

type Executor struct {
    workingDir string
    maxFileSize int64  // Limit file reads to prevent token explosion
}

func NewExecutor(workingDir string) *Executor {
    return &Executor{
        workingDir:  workingDir,
        maxFileSize: 100 * 1024, // 100KB limit
    }
}

func (e *Executor) Execute(name string, args map[string]any) (map[string]any, error) {
    switch name {
    case "read_file":
        return e.readFile(args)
    case "list_directory":
        return e.listDirectory(args)
    case "glob_search":
        return e.globSearch(args)
    default:
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
}

func (e *Executor) readFile(args map[string]any) (map[string]any, error) {
    path := args["path"].(string)
    fullPath := e.resolvePath(path)

    // Security: Validate path is within allowed directory
    if !e.isPathAllowed(fullPath) {
        return map[string]any{"error": "path outside allowed directory"}, nil
    }

    info, err := os.Stat(fullPath)
    if err != nil {
        return map[string]any{"error": err.Error()}, nil
    }

    if info.Size() > e.maxFileSize {
        return map[string]any{
            "error": fmt.Sprintf("file too large (%d bytes, max %d)", info.Size(), e.maxFileSize),
        }, nil
    }

    content, err := os.ReadFile(fullPath)
    if err != nil {
        return map[string]any{"error": err.Error()}, nil
    }

    return map[string]any{
        "path":    fullPath,
        "content": string(content),
        "size":    info.Size(),
    }, nil
}
```

#### 1.3 Integrate with Message Loop

Modify the `sendMessage` function to handle function calls:

```go
func (m model) sendMessage(userMsg string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()

        // Build contents with tool configuration
        config := &genai.GenerateContentConfig{
            Tools: []*genai.Tool{{
                FunctionDeclarations: []*genai.FunctionDeclaration{
                    tools.ReadFileTool,
                    tools.ListDirectoryTool,
                    tools.GlobSearchTool,
                },
            }},
        }

        result, err := m.client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, config)
        if err != nil {
            return responseMsg{err: err}
        }

        // Check for function calls
        functionCalls := result.FunctionCalls()
        if len(functionCalls) > 0 {
            return functionCallMsg{calls: functionCalls}
        }

        return responseMsg{content: result.Text()}
    }
}
```

#### 1.4 Add Function Call UI Feedback

Show users when tools are being used:

```go
type functionCallMsg struct {
    calls []*genai.FunctionCall
}

type functionResultMsg struct {
    results []functionResult
}

// In renderMessages, show tool usage:
if m.executingTools {
    sb.WriteString(infoStyle.Render("Using tools: "))
    for _, name := range m.activeTools {
        sb.WriteString(toolStyle.Render(name + " "))
    }
}
```

#### 1.5 Security Considerations

- **Path Validation**: Restrict file access to current working directory and subdirectories
- **File Size Limits**: Cap readable file size (100KB default) to prevent token explosion
- **Binary Detection**: Skip binary files or return a warning
- **Sensitive Files**: Optionally block `.env`, credentials, private keys

### File Structure After Implementation

```
.
├── main.go
├── internal/
│   └── tools/
│       ├── tools.go       # Tool declarations
│       ├── executor.go    # Tool execution logic
│       └── security.go    # Path validation, limits
├── go.mod
├── go.sum
├── CLAUDE.md
└── PLAN.md
```

---

## Feature 2: Streaming Responses

### Overview

Replace batch response handling with streaming to show text as it's generated. This dramatically improves perceived responsiveness, especially for longer responses.

### Implementation Details

#### 2.1 New Message Types

```go
// Streaming message types
type streamStartMsg struct{}

type streamChunkMsg struct {
    content string
}

type streamEndMsg struct {
    fullContent string
}

type streamErrorMsg struct {
    err error
}
```

#### 2.2 Streaming Command

```go
func (m model) streamMessage(userMsg string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()

        // ... build contents ...

        // Use GenerateContentStream instead of GenerateContent
        stream := m.client.Models.GenerateContentStream(ctx, "gemini-2.0-flash", contents, nil)

        // Return a command that will send chunks
        return streamStartMsg{}
    }
}
```

#### 2.3 Background Streaming with Channel

Since bubbletea uses a message-based architecture, we need to stream via a goroutine:

```go
func (m model) streamMessage(userMsg string) tea.Cmd {
    return func() tea.Msg {
        return streamStartMsg{userMsg: userMsg}
    }
}

// In Update, handle streamStartMsg by spawning goroutine
case streamStartMsg:
    m.streaming = true
    m.streamBuffer = ""
    return m, m.startStreaming(msg.userMsg)

func (m model) startStreaming(userMsg string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        contents := m.buildContents(userMsg)

        var fullText strings.Builder

        for resp, err := range m.client.Models.GenerateContentStream(ctx, "gemini-2.0-flash", contents, nil) {
            if err != nil {
                return streamErrorMsg{err: err}
            }
            chunk := resp.Text()
            fullText.WriteString(chunk)
            // Send chunk to program - need to use program.Send()
        }

        return streamEndMsg{fullContent: fullText.String()}
    }
}
```

#### 2.4 Program Reference for Async Updates

Store program reference for sending messages from goroutines:

```go
type model struct {
    // ... existing fields ...
    program      *tea.Program
    streaming    bool
    streamBuffer string
}

// In main():
m := initialModel(client)
p := tea.NewProgram(m, tea.WithAltScreen())
m.program = p  // Store reference
```

#### 2.5 Incremental Markdown Rendering

For streaming, we need to handle partial markdown gracefully:

```go
func (m model) renderStreamingContent(partial string) string {
    // For incomplete markdown, render what we can
    // Fall back to plain text for incomplete code blocks
    rendered, err := m.mdRenderer.Render(partial)
    if err != nil {
        return partial  // Show raw text if markdown parsing fails
    }
    return rendered
}
```

### UI Considerations

- Show a streaming indicator (blinking cursor or spinner)
- Update viewport content on each chunk
- Auto-scroll to bottom during streaming
- Disable input during streaming (or queue input)

---

## Feature 3: Thinking Mode

### Overview

Enable Gemini's extended thinking capability for complex reasoning tasks. The thinking process helps with coding problems, debugging, mathematical reasoning, and multi-step analysis.

### Implementation Details

#### 3.1 Add Thinking Toggle

Add a keyboard shortcut or command to toggle thinking mode:

```go
type model struct {
    // ... existing fields ...
    thinkingEnabled bool
    thinkingBudget  int  // 0 = disabled, -1 = dynamic, or specific token count
}

// Handle toggle in Update
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+t":
        m.thinkingEnabled = !m.thinkingEnabled
        return m, nil
    }
```

#### 3.2 Configure Thinking in Requests

```go
func (m model) buildConfig() *genai.GenerateContentConfig {
    config := &genai.GenerateContentConfig{}

    if m.thinkingEnabled {
        config.ThinkingConfig = &genai.ThinkingConfig{
            IncludeThoughts: true,
            ThinkingBudget:  m.thinkingBudget,  // -1 for dynamic
        }
    }

    return config
}
```

#### 3.3 Display Thinking Process

Show the model's thinking in a collapsible or dimmed section:

```go
type message struct {
    role     string
    content  string
    thinking string  // New field for thought process
}

func (m model) renderMessages() string {
    // ...
    for _, msg := range m.messages {
        if msg.role == "assistant" {
            // Show thinking if present
            if msg.thinking != "" {
                sb.WriteString(thinkingStyle.Render("Thinking:\n"))
                sb.WriteString(dimStyle.Render(msg.thinking))
                sb.WriteString("\n\n")
            }
            // Show response
            sb.WriteString(assistantStyle.Render("Gemini:"))
            // ...
        }
    }
}
```

#### 3.4 Model Selection

Thinking works best with specific models. Add model selection:

```go
const (
    ModelFlash   = "gemini-2.0-flash"
    ModelPro     = "gemini-2.5-pro"       // Better for thinking
    ModelFlash25 = "gemini-2.5-flash"     // Good balance
)

type model struct {
    // ...
    currentModel string
}

// Ctrl+M to cycle models
case "ctrl+m":
    m.currentModel = m.nextModel()
```

### UI Additions

- Status bar showing: `[Model: gemini-2.5-pro] [Thinking: ON]`
- Visual indicator when model is "thinking" vs generating response
- Collapsible thinking section (toggle with key)

---

## Implementation Order

### Phase 1: Foundation
1. Restructure code into packages (`internal/tools/`, `internal/ui/`)
2. Add configuration system for settings persistence
3. Implement basic streaming (biggest UX improvement)

### Phase 2: Tool Use
4. Define tool declarations
5. Implement tool executor with security
6. Add function call handling loop
7. Add tool usage UI feedback

### Phase 3: Thinking Mode
8. Add thinking configuration
9. Handle thinking in responses
10. Add model selection
11. Add status bar with current settings

### Phase 4: Polish
12. Add command palette (`:` prefix for commands)
13. Persist settings to config file
14. Add help screen (`?` key)
15. Error recovery and retry logic

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/bmatcuk/doublestar/v4  // For glob patterns
)
```

---

## Configuration File

Future: Add `~/.config/gemini-tui/config.yaml`:

```yaml
model: gemini-2.5-flash
thinking:
  enabled: false
  budget: -1  # dynamic
tools:
  enabled: true
  max_file_size: 102400
  allowed_paths:
    - "."
theme: dark
```

---

## Testing Considerations

- Mock Gemini client for unit tests
- Test tool executor with filesystem fixtures
- Test streaming with mock iterators
- Integration tests with real API (optional, requires key)
