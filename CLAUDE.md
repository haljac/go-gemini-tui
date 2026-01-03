# CLAUDE.md

This file provides guidance for Claude Code when working with this project.

**Important**: Before implementing new features, read [PLAN.md](./PLAN.md) for the detailed feature roadmap and implementation specifications.

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
├── CLAUDE.md                  # This file
└── PLAN.md                    # Feature roadmap and implementation plan
```

## Current Features

### File System Tools (Implemented)
Gemini can use these tools to interact with the filesystem:
- **read_file**: Read contents of files (100KB limit, text files only)
- **list_directory**: List files and directories
- **glob_search**: Find files matching patterns (e.g., `**/*.go`)

Security: Tools are restricted to the working directory and subdirectories.

## Feature Roadmap

See [PLAN.md](./PLAN.md) for detailed implementation plans. Remaining features:

1. ~~**File System Tool Use**~~ - Done
2. **Streaming Responses** - Show responses as they're generated for better UX
3. **Thinking Mode** - Enable extended reasoning for complex coding/math tasks

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
- Async message sending via `tea.Cmd`

### Function Calling Flow
1. User sends message -> `sendMessage()` called
2. If Gemini returns function calls -> `functionCallMsg` sent to Update
3. Update executes tools via `toolExecutor.Execute()`
4. Function results sent back to Gemini via `continueWithFunctionResults()`
5. Loop continues until Gemini returns text response

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
