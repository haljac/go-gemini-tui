# CLAUDE.md

This file provides guidance for Claude Code when working with this project.

## Project Overview

A terminal user interface (TUI) for interacting with Google's Gemini AI model. Built with Go using the Bubbletea framework for the TUI and Google's official genai SDK.

## Tech Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) - follows the Elm architecture (Model, Update, View)
- **UI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) - textarea, viewport
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **AI SDK**: [Google GenAI](https://github.com/googleapis/go-genai) - unified SDK for Gemini

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
├── main.go          # Application entry point and TUI implementation
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
└── CLAUDE.md        # This file
```

## Architecture Notes

The application follows the Elm architecture pattern:

1. **Model**: Holds application state (messages, viewport, textarea, client)
2. **Update**: Handles events (keypresses, API responses, window resizes)
3. **View**: Renders the UI as a string

Key components:
- `textarea`: User input area
- `viewport`: Scrollable message history
- Async message sending via `tea.Cmd`

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
