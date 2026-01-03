# CLAUDE.md

This file provides guidance for Claude Code when working with this project.

## Key Files to Read

- [README.md](./README.md) - User-facing documentation, features, and usage instructions
- [PLAN.md](./PLAN.md) - Detailed implementation specifications and architecture decisions

## Project Overview

A terminal user interface (TUI) for interacting with Google's Gemini AI model. Built with Go using the Bubbletea framework for the TUI and Google's official genai SDK.

## Tech Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) - follows the Elm architecture (Model, Update, View)
- **UI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) - textarea, viewport
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Markdown Rendering**: [Glamour](https://github.com/charmbracelet/glamour) - styled markdown for terminal
- **AI SDK**: [Google GenAI](https://github.com/googleapis/go-genai) - unified SDK for Gemini
- **Glob Matching**: [doublestar](https://github.com/bmatcuk/doublestar) - for `**` glob patterns

## Build Commands

```bash
# Build the application
go build -o gemini-tui .

# Run directly
go run .

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

## Environment Variables

- `GOOGLE_API_KEY` - Required. Your Gemini API key from https://aistudio.google.com/apikey

## Project Structure

```
.
├── main.go                    # Application entry point and TUI implementation
├── internal/
│   └── tools/
│       ├── tools.go           # Tool declarations (read_file, list_directory, glob_search)
│       └── executor.go        # Tool execution with security constraints
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── README.md                  # User documentation
├── CLAUDE.md                  # This file (development guidance)
└── PLAN.md                    # Feature roadmap and implementation plan
```

## Current Features

### File System Tools (Implemented)
Gemini can use these tools to interact with the filesystem:
- **read_file**: Read contents of files (100KB limit, text files only)
- **list_directory**: List files and directories
- **glob_search**: Find files matching patterns (e.g., `**/*.go`)

Security: Tools are restricted to the working directory and subdirectories.

### Streaming Responses (Implemented)
Responses stream in real-time as Gemini generates them:
- Text appears incrementally as it's generated
- Works seamlessly with tool use (tools execute, then response streams)
- Visual indicator ("...") while streaming
- Markdown rendered after streaming completes

### Thinking Mode (Implemented)
Extended reasoning for complex tasks:
- **Ctrl+T**: Toggle thinking mode on/off
- **Ctrl+G**: Cycle through models (gemini-2.0-flash, gemini-2.5-flash, gemini-2.5-pro)
- **Ctrl+H**: Toggle display of thinking content
- Shows model's reasoning process before final response
- Status bar displays current model and thinking state

## Feature Roadmap

All planned features are complete! See [PLAN.md](./PLAN.md) for implementation details.

1. ~~**File System Tool Use**~~ - Done
2. ~~**Streaming Responses**~~ - Done
3. ~~**Thinking Mode**~~ - Done

## Architecture Notes

The application follows the Elm architecture pattern:

1. **Model**: Holds application state (messages, viewport, textarea, client, toolExecutor)
2. **Update**: Handles events (keypresses, API responses, function calls, window resizes)
3. **View**: Renders the UI as a string

Key components:
- `textarea`: User input area
- `viewport`: Scrollable message history
- `mdRenderer`: Glamour markdown renderer (auto-adapts to terminal width)
- `toolExecutor`: Handles file system tool execution with security
- `thinkingEnabled`: Toggle for extended reasoning mode
- `currentModel`: Active Gemini model (flash-2.0, flash-2.5, pro-2.5)
- Async message sending via `tea.Cmd`

### Streaming & Function Calling Flow
1. User sends message -> `sendMessage()` starts streaming via goroutine
2. Chunks arrive via channel -> `streamChunkMsg` updates UI incrementally
3. If function calls detected -> `streamFunctionCallMsg` executes tools
4. Tool results sent back -> streaming continues via `continueWithFunctionResults()`
5. When done -> `streamDoneMsg` finalizes message with markdown rendering

## Code Style

- Use standard Go formatting (`go fmt`)
- Keep functions focused and small
- Handle errors explicitly
- Use meaningful variable names

## When Modifying

- Run `go build` to verify compilation
- Test with a real API key before committing
- Keep the UI responsive (use `tea.Cmd` for async operations)
- Consider terminal size constraints when adding UI elements
