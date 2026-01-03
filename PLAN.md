# Feature Implementation Plan

This document outlines the implementation plan for Gemini TUI—a terminal-based coding agent powered by Google's Gemini AI.

## Project Vision

Gemini TUI is a coding assistant that runs in your terminal. It can read, write, and edit files in your project, helping you write code, debug issues, refactor, and understand codebases. Think of it as a Gemini-powered alternative to Claude Code.

## Feature Overview

| Feature | Priority | Complexity | Status |
|---------|----------|------------|--------|
| 1. File System Read Tools | High | Medium | **DONE** |
| 2. Streaming Responses | High | Medium | **DONE** |
| 3. Thinking Mode | Medium | Low | **DONE** |
| 4. Distribution & Installation | High | Medium | **DONE** |
| 5. File System Write Tools | High | Medium | **DONE** |
| 6. Gemini 3 Models | Medium | Low | **DONE** |
| 7. Configuration System | High | Medium | **IN PROGRESS** |

---

## Feature 1: File System Tool Use [COMPLETED]

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

## Feature 4: Distribution & Installation

### Overview

Enable users to install gemini-tui with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/haljac/go-gemini-tui/master/install.sh | bash
```

This requires building binaries for multiple platforms, hosting them on GitHub Releases, and providing an install script that detects the user's platform and downloads the appropriate binary.

### Prerequisites

- Git remote configured: `origin -> git@github.com:haljac/go-gemini-tui.git`
- GitHub CLI (`gh`) authenticated with repo access
- Go toolchain for cross-compilation

### Implementation Steps

#### 4.1 Create Makefile for Cross-Compilation

Create a `Makefile` with targets for building binaries for all supported platforms:

**Supported Platforms:**
| OS | Architecture | Binary Name |
|----|--------------|-------------|
| Linux | amd64 | `gemini-tui-linux-amd64` |
| Linux | arm64 | `gemini-tui-linux-arm64` |
| macOS | amd64 (Intel) | `gemini-tui-darwin-amd64` |
| macOS | arm64 (Apple Silicon) | `gemini-tui-darwin-arm64` |

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME = gemini-tui
DIST_DIR = dist

.PHONY: all clean build-all release

all: build-all

clean:
    rm -rf $(DIST_DIR)

build-all: clean
    mkdir -p $(DIST_DIR)
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
    GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
    GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
```

#### 4.2 Create Install Script

Create `install.sh` in the repository root. The script will:

1. Detect OS (Linux/macOS) and architecture (amd64/arm64)
2. Fetch the latest release tag from GitHub API
3. Download the appropriate binary from GitHub Releases
4. Install to `~/.local/bin` (user) or `/usr/local/bin` (with sudo)
5. Verify the binary works

**Key Script Features:**
- No dependencies beyond `curl` and standard Unix tools
- Graceful error handling with helpful messages
- Support for custom install directory via environment variable
- Checksum verification (optional enhancement)

```bash
#!/bin/bash
set -euo pipefail

REPO="haljac/go-gemini-tui"
BINARY_NAME="gemini-tui"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

# Download binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/${BINARY_NAME}-${OS}-${ARCH}"
echo "Downloading $BINARY_NAME $LATEST for $OS/$ARCH..."
curl -fsSL "$DOWNLOAD_URL" -o "/tmp/$BINARY_NAME"
chmod +x "/tmp/$BINARY_NAME"

# Install
mkdir -p "$INSTALL_DIR"
mv "/tmp/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
echo "Installed to $INSTALL_DIR/$BINARY_NAME"

# Verify
if command -v "$BINARY_NAME" &>/dev/null; then
    echo "Success! Run '$BINARY_NAME' to start."
else
    echo "Add $INSTALL_DIR to your PATH: export PATH=\"\$PATH:$INSTALL_DIR\""
fi
```

#### 4.3 Create GitHub Release

Use `gh` CLI to create releases. This can be done manually or automated via CI.

**Manual Release Process:**

```bash
# 1. Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. Build all binaries
make build-all

# 3. Create GitHub release with binaries
gh release create v1.0.0 \
    --title "v1.0.0" \
    --notes "Initial release with file tools, streaming, and thinking mode" \
    dist/*
```

#### 4.4 Version Embedding (Optional Enhancement)

Add version info to the binary via ldflags:

```go
// main.go
var version = "dev"

func main() {
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Println("gemini-tui", version)
        os.Exit(0)
    }
    // ... rest of main
}
```

Build with: `go build -ldflags "-X main.version=v1.0.0"`

### Files to Create

| File | Purpose |
|------|---------|
| `Makefile` | Cross-compilation and release targets |
| `install.sh` | User-facing installation script |

### Release Checklist

1. [ ] Ensure all tests pass
2. [ ] Update version number (if hardcoded anywhere)
3. [ ] Create and push git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
4. [ ] Build binaries: `make build-all`
5. [ ] Create GitHub release: `gh release create vX.Y.Z dist/* --title "vX.Y.Z" --notes "..."`
6. [ ] Test install script: `curl -fsSL https://raw.githubusercontent.com/haljac/go-gemini-tui/master/install.sh | bash`

### Manual Steps Required

These steps cannot be automated by Claude Code:

1. **Push git tags** - Requires git push access (Claude can create tags locally but pushing may require user action if SSH keys aren't configured)
2. **Verify installation** - User should test the install script on their own machine after release

### Future Enhancements

- **GoReleaser**: Automate the entire release process with `.goreleaser.yaml`
- **GitHub Actions**: CI/CD pipeline to build and release on tag push
- **Homebrew Formula**: `brew install haljac/tap/gemini-tui`
- **Checksums**: Generate and verify SHA256 checksums for binaries
- **Windows Support**: Add Windows binaries (would need .exe suffix handling in install script)

---

## Feature 7: Configuration System [IN PROGRESS]

### Overview

Add persistent configuration using the XDG Base Directory specification with TOML format. Users can configure model choice, thinking mode, and UI preferences that persist across sessions.

### Configuration Location

Following XDG Base Directory spec:
- **Config file**: `$XDG_CONFIG_HOME/gemini-tui/config.toml` (defaults to `~/.config/gemini-tui/config.toml`)
- **Fallback**: If `XDG_CONFIG_HOME` is not set, use `~/.config`

### Config File Format (TOML)

```toml
# Gemini TUI Configuration

# Model to use (gemini-2.0-flash, gemini-2.5-flash, gemini-2.5-pro, gemini-3-flash-preview, gemini-3-pro-preview)
model = "gemini-2.0-flash"

[thinking]
# Enable thinking mode for complex reasoning
enabled = false
# Show thinking process in UI
show = true
```

### Configurable Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `model` | string | `"gemini-2.0-flash"` | Active Gemini model |
| `thinking.enabled` | bool | `false` | Whether thinking mode is on |
| `thinking.show` | bool | `true` | Whether to display thinking in UI |

### Implementation Steps

#### 7.1 Add TOML dependency

Add the BurntSushi/toml package to go.mod:

```bash
go get github.com/BurntSushi/toml
```

#### 7.2 Create config package

Create `internal/config/config.go`:

```go
package config

import (
    "os"
    "path/filepath"

    "github.com/BurntSushi/toml"
)

type Config struct {
    Model    string         `toml:"model"`
    Thinking ThinkingConfig `toml:"thinking"`
}

type ThinkingConfig struct {
    Enabled bool `toml:"enabled"`
    Show    bool `toml:"show"`
}

func DefaultConfig() *Config {
    return &Config{
        Model: "gemini-2.0-flash",
        Thinking: ThinkingConfig{
            Enabled: false,
            Show:    true,
        },
    }
}

func configDir() string {
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "gemini-tui")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "gemini-tui")
}

func configPath() string {
    return filepath.Join(configDir(), "config.toml")
}
```

#### 7.3 Implement Load function

```go
func Load() (*Config, error) {
    cfg := DefaultConfig()

    path := configPath()
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return cfg, nil // Return defaults if no config file
    }

    _, err := toml.DecodeFile(path, cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    return cfg, nil
}
```

#### 7.4 Implement Save function

```go
func (c *Config) Save() error {
    dir := configDir()
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create config dir: %w", err)
    }

    f, err := os.Create(configPath())
    if err != nil {
        return fmt.Errorf("failed to create config file: %w", err)
    }
    defer f.Close()

    encoder := toml.NewEncoder(f)
    return encoder.Encode(c)
}
```

#### 7.5 Integrate with main.go

Update model struct and initialization:

```go
type model struct {
    // ... existing fields ...
    config *config.Config
}

func initialModel(client *genai.Client, executor *tools.Executor) model {
    cfg, err := config.Load()
    if err != nil {
        // Log warning, use defaults
        cfg = config.DefaultConfig()
    }

    return model{
        // ... existing initialization ...
        config:          cfg,
        currentModel:    cfg.Model,
        thinkingEnabled: cfg.Thinking.Enabled,
        showThinking:    cfg.Thinking.Show,
    }
}
```

#### 7.6 Save on toggle

Update keyboard handlers to save config on change:

```go
case "ctrl+t":
    m.thinkingEnabled = !m.thinkingEnabled
    m.config.Thinking.Enabled = m.thinkingEnabled
    m.config.Save() // Persist change
    return m, nil

case "ctrl+g":
    m.currentModel = m.nextModel()
    m.config.Model = m.currentModel
    m.config.Save() // Persist change
    return m, nil

case "ctrl+h":
    m.showThinking = !m.showThinking
    m.config.Thinking.Show = m.showThinking
    m.config.Save() // Persist change
    return m, nil
```

### File Structure After Implementation

```
.
├── main.go
├── internal/
│   ├── config/
│   │   └── config.go      # Configuration loading/saving
│   └── tools/
│       ├── tools.go
│       └── executor.go
├── go.mod
└── ...
```

### Implementation Checklist

- [ ] 7.1 Add `github.com/BurntSushi/toml` dependency
- [ ] 7.2 Create `internal/config/config.go` with Config struct and defaults
- [ ] 7.3 Implement `Load()` function with XDG path resolution
- [ ] 7.4 Implement `Save()` function
- [ ] 7.5 Integrate config loading in `main.go` initialization
- [ ] 7.6 Save config on each toggle (Ctrl+T, Ctrl+G, Ctrl+H)
- [ ] 7.7 Test: verify config persists across restarts
- [ ] 7.8 Update README.md with configuration documentation

---

## Implementation Order

### Phase 1: Foundation [DONE]
1. ~~Restructure code into packages (`internal/tools/`, `internal/ui/`)~~
2. ~~Add configuration system for settings persistence~~
3. ~~Implement basic streaming (biggest UX improvement)~~

### Phase 2: Tool Use [DONE]
4. ~~Define tool declarations~~
5. ~~Implement tool executor with security~~
6. ~~Add function call handling loop~~
7. ~~Add tool usage UI feedback~~

### Phase 3: Thinking Mode [DONE]
8. ~~Add thinking configuration~~
9. ~~Handle thinking in responses~~
10. ~~Add model selection~~
11. ~~Add status bar with current settings~~

### Phase 4: Distribution [DONE]
12. ~~Create `Makefile` with cross-compilation targets~~
13. ~~Create `install.sh` script~~
14. ~~Add version embedding to binary (`--version` flag)~~
15. ~~Create initial GitHub release with `gh` CLI~~
16. ~~Test installation via curl pipe~~

### Phase 5: Coding Agent [DONE]
17. ~~Add `write_file` tool for creating/overwriting files~~
18. ~~Add `edit_file` tool for surgical string replacement~~
19. ~~Add `create_directory` tool~~
20. ~~Update system prompt for coding agent behavior~~
21. ~~Add Gemini 3 preview models~~

### Phase 6: Configuration System [IN PROGRESS]
22. [ ] Add `github.com/BurntSushi/toml` dependency
23. [ ] Create `internal/config/config.go` with Config struct and defaults
24. [ ] Implement `Load()` function with XDG path resolution
25. [ ] Implement `Save()` function
26. [ ] Integrate config loading in `main.go` initialization
27. [ ] Save config on each toggle (Ctrl+T, Ctrl+G, Ctrl+H)
28. [ ] Test: verify config persists across restarts
29. [ ] Update README.md with configuration documentation

### Phase 7: Polish (Future)
30. Add command palette (`:` prefix for commands)
31. Add help screen (`?` key)
32. Error recovery and retry logic
33. Bash/shell command execution tool
34. Git integration tools (status, diff, commit)
35. GitHub Actions for automated releases

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

Configuration uses XDG Base Directory spec with TOML format.

**Location**: `$XDG_CONFIG_HOME/gemini-tui/config.toml` (defaults to `~/.config/gemini-tui/config.toml`)

```toml
# Gemini TUI Configuration

# Model to use
model = "gemini-2.0-flash"

[thinking]
# Enable thinking mode for complex reasoning
enabled = false
# Show thinking process in UI
show = true
```

See **Feature 7: Configuration System** for implementation details.

---

## Testing Considerations

- Mock Gemini client for unit tests
- Test tool executor with filesystem fixtures
- Test streaming with mock iterators
- Integration tests with real API (optional, requires key)
