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

// AllTools returns all available tool declarations
func AllTools() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		ReadFileTool,
		ListDirectoryTool,
		GlobSearchTool,
	}
}
