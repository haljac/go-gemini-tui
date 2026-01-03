# CLAUDE.md

This file provides guidance for Claude Code when working with this project.

## Key Files to Read

- [README.md](./README.md) - User-facing documentation, installation, and usage
- [PLAN.md](./PLAN.md) - Implementation history and future roadmap

## Project Overview

**Gemini TUI** is a terminal-based coding agent powered by Google's Gemini AI. It allows users to interact with Gemini in a terminal interface where the AI can read, write, and edit files in the user's project directory.

**Primary use case**: A coding assistant that can help users write code, debug issues, refactor, and understand codebases—similar to Claude Code but using Gemini models.

## Tech Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) - Elm architecture (Model, Update, View)
- **UI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) - textarea, viewport
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Markdown**: [Glamour](https://github.com/charmbracelet/glamour) - terminal markdown rendering
- **AI SDK**: [Google GenAI](https://github.com/googleapis/go-genai) - Gemini API client
- **Glob**: [doublestar](https://github.com/bmatcuk/doublestar) - `**` glob patterns

## Build Commands

```bash
# Build for current platform
make build

# Build for all platforms (linux/darwin × amd64/arm64)
make build-all

# Create a new release
make release V=v1.0.0

# Run directly
go run .

# Run tests
go test ./...
```

## Environment Variables

- `GOOGLE_API_KEY` - Required. Gemini API key from https://aistudio.google.com/apikey

## Project Structure

```
.
├── main.go                    # Application entry point and TUI logic
├── internal/
│   └── tools/
│       ├── tools.go           # Tool declarations for Gemini
│       └── executor.go        # Tool execution with security constraints
├── Makefile                   # Build and release automation
├── install.sh                 # Curl-pipe installation script
├── go.mod / go.sum            # Go module files
├── README.md                  # User documentation
├── CLAUDE.md                  # This file
└── PLAN.md                    # Implementation roadmap
```

## Available Tools

The coding agent has these filesystem tools:

### Reading Tools
- **read_file**: Read file contents (100KB limit, text files only)
- **list_directory**: List files and directories
- **glob_search**: Find files matching patterns (e.g., `**/*.go`)

### Writing Tools
- **write_file**: Create new files or overwrite existing files
- **edit_file**: Make surgical edits by string replacement (old_string must be unique)
- **create_directory**: Create directories (including parents)

**Security**: All tools are restricted to the working directory and subdirectories.

## Available Models

Models are cycled with `Ctrl+G`:
- `gemini-2.0-flash` - Fast, default
- `gemini-2.5-flash` - Balanced
- `gemini-2.5-pro` - Powerful reasoning
- `gemini-3-flash-preview` - Latest multimodal
- `gemini-3-pro-preview` - Most capable

## Architecture

The application follows the Elm architecture:

1. **Model**: Application state (messages, viewport, textarea, client, toolExecutor)
2. **Update**: Event handling (keypresses, API responses, function calls)
3. **View**: Renders UI as a string

### Key Flow: Streaming with Tool Calls
1. User sends message → `sendMessage()` starts streaming
2. Chunks arrive → `streamChunkMsg` updates UI
3. Function calls detected → `streamFunctionCallMsg` executes tools
4. Tool results sent → streaming continues via `continueWithFunctionResults()`
5. Complete → `streamDoneMsg` finalizes with markdown rendering

## Code Style

- Standard Go formatting (`go fmt`)
- Small, focused functions
- Explicit error handling
- Meaningful variable names

## When Modifying

- Run `go build` to verify compilation
- Test changes with a real API key
- Keep UI responsive (use `tea.Cmd` for async)
- Consider terminal size constraints
- Update this file and PLAN.md if adding features
