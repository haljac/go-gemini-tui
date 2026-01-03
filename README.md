# Gemini TUI

A terminal-based coding agent powered by Google's Gemini AI. Built with Go using the Bubbletea framework.

Gemini TUI can read, write, and edit files in your project, making it a powerful assistant for writing code, debugging, refactoring, and understanding codebases.

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/haljac/go-gemini-tui/master/install.sh | bash
```

This installs the latest release to `~/.local/bin`. You can customize the install location:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/haljac/go-gemini-tui/master/install.sh | bash
```

### Build from Source

Prerequisites: Go 1.21 or later

```bash
git clone https://github.com/haljac/go-gemini-tui.git
cd go-gemini-tui
go build -o gemini-tui .
```

## Setup

1. Get an API key from [Google AI Studio](https://aistudio.google.com/apikey)

2. Set your API key:

```bash
export GOOGLE_API_KEY="your-api-key-here"
```

3. Run in your project directory:

```bash
cd /path/to/your/project
gemini-tui
```

## Features

- **Coding Agent** - Gemini can read, write, and edit files in your project
- **Streaming Responses** - See responses as they're generated in real-time
- **Thinking Mode** - Enable extended reasoning for complex tasks
- **Multiple Models** - Switch between Gemini 2.0, 2.5, and 3.0 models
- **Markdown Rendering** - Responses rendered with syntax highlighting

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+T` | Toggle thinking mode |
| `Ctrl+G` | Cycle through models |
| `Ctrl+H` | Toggle display of thinking content |
| `Esc` / `Ctrl+C` | Quit |

## File System Tools

Gemini has full access to read and write files within your project directory:

### Reading
- **read_file** - Read file contents (up to 100KB)
- **list_directory** - List files and directories
- **glob_search** - Find files matching patterns (e.g., `**/*.go`)

### Writing
- **write_file** - Create new files or overwrite existing files
- **edit_file** - Make surgical edits by replacing specific strings
- **create_directory** - Create directories

### Example Prompts

```
"Create a new Go file that implements a binary search function"
"Read main.go and add error handling to the HTTP handler"
"Find all TypeScript files and list their exports"
"Refactor this function to use async/await"
"Add unit tests for the Calculator class"
```

### Security

All file operations are restricted to the current working directory and its subdirectories. The agent cannot access files outside your project.

## Models

| Model | Description |
|-------|-------------|
| `gemini-2.0-flash` | Fast responses, good for most tasks (default) |
| `gemini-2.5-flash` | Balanced speed and capability |
| `gemini-2.5-pro` | Powerful reasoning with adaptive thinking |
| `gemini-3-flash-preview` | Latest multimodal model with strong reasoning |
| `gemini-3-pro-preview` | Most capable, optimized for complex agentic workflows |

Use `Ctrl+G` to cycle between models. For complex coding tasks, try `gemini-2.5-pro` or `gemini-3-pro-preview` with thinking mode enabled (`Ctrl+T`).

> **Note**: Gemini 3 models are currently in preview. The `gemini-3-pro-preview` model may not have a free tier.

## Thinking Mode

Enable thinking mode with `Ctrl+T` to see Gemini's reasoning process. This is especially useful for:

- Complex refactoring
- Debugging tricky issues
- Architectural decisions
- Multi-file changes

Toggle visibility of thinking content with `Ctrl+H`.

## Project Structure

```
.
├── main.go                 # Application entry point and TUI logic
├── internal/
│   └── tools/
│       ├── tools.go        # Tool declarations for Gemini
│       └── executor.go     # Tool execution with security
├── Makefile                # Build and release targets
├── install.sh              # Installation script
├── go.mod
├── go.sum
├── README.md               # This file
├── CLAUDE.md               # Development guidelines
└── PLAN.md                 # Feature implementation details
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT
