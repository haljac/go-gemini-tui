# Gemini TUI

A terminal user interface for interacting with Google's Gemini AI. Built with Go using the Bubbletea framework.

## Features

- **Conversational AI** - Chat with Gemini directly from your terminal
- **Streaming Responses** - See responses as they're generated in real-time
- **File System Tools** - Gemini can read files, list directories, and search for files in your project
- **Thinking Mode** - Enable extended reasoning for complex tasks
- **Multiple Models** - Switch between gemini-2.0-flash, gemini-2.5-flash, and gemini-2.5-pro
- **Markdown Rendering** - AI responses are rendered with syntax highlighting and formatting

## Installation

### Prerequisites

- Go 1.21 or later
- A Google API key from [Google AI Studio](https://aistudio.google.com/apikey)

### Build from source

```bash
git clone https://github.com/haljac/gemini-tui.git
cd gemini-tui
go build -o gemini-tui .
```

## Usage

1. Set your API key:

```bash
export GOOGLE_API_KEY="your-api-key-here"
```

2. Run the application:

```bash
./gemini-tui
```

Or run directly with Go:

```bash
go run .
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+T` | Toggle thinking mode |
| `Ctrl+G` | Cycle through models |
| `Ctrl+H` | Toggle display of thinking content |
| `Esc` / `Ctrl+C` | Quit |

## File System Tools

Gemini can interact with your local filesystem through built-in tools:

- **read_file** - Read the contents of text files (up to 100KB)
- **list_directory** - List files and directories at a path
- **glob_search** - Find files matching patterns (e.g., `**/*.go`)

Example prompts:
- "What files are in this project?"
- "Read the main.go file and explain what it does"
- "Find all Go files in this directory"

### Security

File system access is restricted to the current working directory and its subdirectories. Binary files are automatically detected and skipped.

## Models

| Model | Description |
|-------|-------------|
| `gemini-2.0-flash` | Fast responses, good for most tasks (default) |
| `gemini-2.5-flash` | Balanced speed and capability |
| `gemini-2.5-pro` | Most capable, best for complex reasoning |

Use `Ctrl+G` to cycle between models during a session.

## Thinking Mode

Enable thinking mode with `Ctrl+T` to see Gemini's reasoning process before its final response. This is useful for:

- Complex coding problems
- Multi-step analysis
- Debugging assistance
- Mathematical reasoning

Toggle visibility of thinking content with `Ctrl+H`.

## Project Structure

```
.
├── main.go                 # Application entry point and TUI logic
├── internal/
│   └── tools/
│       ├── tools.go        # Tool declarations for Gemini
│       └── executor.go     # Tool execution with security
├── go.mod
├── go.sum
├── CLAUDE.md               # Development guidelines
└── PLAN.md                 # Feature implementation details
```

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Google GenAI SDK](https://github.com/googleapis/go-genai) - Gemini API client
- [doublestar](https://github.com/bmatcuk/doublestar) - Glob pattern matching

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
