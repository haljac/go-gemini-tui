package tools

import "google.golang.org/genai"

// Tool declarations for file system operations

var ReadFileTool = &genai.FunctionDeclaration{
	Name:        "read_file",
	Description: "Read the contents of a file at the given path. Use this to examine source code, configuration files, documentation, or any text file. Returns the file contents along with metadata.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"path": {
				Type:        genai.TypeString,
				Description: "The file path to read (absolute or relative to working directory)",
			},
		},
		Required: []string{"path"},
	},
}

var ListDirectoryTool = &genai.FunctionDeclaration{
	Name:        "list_directory",
	Description: "List files and directories at the given path. Returns names with type indicators (directories end with /). Useful for exploring project structure.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"path": {
				Type:        genai.TypeString,
				Description: "The directory path to list. Use '.' or empty for current directory.",
			},
		},
		Required: []string{},
	},
}

var GlobSearchTool = &genai.FunctionDeclaration{
	Name:        "glob_search",
	Description: "Find files matching a glob pattern. Useful for finding all files of a certain type. Examples: '*.go' for Go files in current dir, '**/*.go' for all Go files recursively, 'src/**/*.ts' for TypeScript files in src.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"pattern": {
				Type:        genai.TypeString,
				Description: "Glob pattern to match (e.g., '*.go', '**/*.ts', 'src/**/*.js')",
			},
		},
		Required: []string{"pattern"},
	},
}

var WriteFileTool = &genai.FunctionDeclaration{
	Name:        "write_file",
	Description: "Write content to a file, creating it if it doesn't exist or overwriting if it does. Use this to create new files or completely replace file contents. For partial edits, use edit_file instead.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"path": {
				Type:        genai.TypeString,
				Description: "The file path to write to (relative to working directory)",
			},
			"content": {
				Type:        genai.TypeString,
				Description: "The content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	},
}

var EditFileTool = &genai.FunctionDeclaration{
	Name:        "edit_file",
	Description: "Edit an existing file by replacing a specific string with new content. The old_string must match exactly (including whitespace and indentation). Use this for surgical edits to existing files. For creating new files or full rewrites, use write_file.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"path": {
				Type:        genai.TypeString,
				Description: "The file path to edit (relative to working directory)",
			},
			"old_string": {
				Type:        genai.TypeString,
				Description: "The exact string to find and replace (must match exactly, including whitespace)",
			},
			"new_string": {
				Type:        genai.TypeString,
				Description: "The string to replace old_string with",
			},
		},
		Required: []string{"path", "old_string", "new_string"},
	},
}

var CreateDirectoryTool = &genai.FunctionDeclaration{
	Name:        "create_directory",
	Description: "Create a new directory (and any necessary parent directories). Use this before writing files to new directories.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"path": {
				Type:        genai.TypeString,
				Description: "The directory path to create (relative to working directory)",
			},
		},
		Required: []string{"path"},
	},
}

// AllTools returns all available tool declarations
func AllTools() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		ReadFileTool,
		ListDirectoryTool,
		GlobSearchTool,
		WriteFileTool,
		EditFileTool,
		CreateDirectoryTool,
	}
}
