package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Executor handles tool execution with security constraints
type Executor struct {
	workingDir  string
	maxFileSize int64
	maxResults  int
}

// NewExecutor creates a new tool executor rooted at the given directory
func NewExecutor(workingDir string) (*Executor, error) {
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}

	return &Executor{
		workingDir:  absDir,
		maxFileSize: 100 * 1024, // 100KB limit
		maxResults:  100,        // Max glob results
	}, nil
}

// Execute runs a tool by name with the given arguments
func (e *Executor) Execute(name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "read_file":
		return e.readFile(args)
	case "list_directory":
		return e.listDirectory(args)
	case "glob_search":
		return e.globSearch(args)
	case "write_file":
		return e.writeFile(args)
	case "edit_file":
		return e.editFile(args)
	case "create_directory":
		return e.createDirectory(args)
	default:
		return map[string]any{"error": fmt.Sprintf("unknown tool: %s", name)}, nil
	}
}

// readFile reads the contents of a file
func (e *Executor) readFile(args map[string]any) (map[string]any, error) {
	pathArg, ok := args["path"].(string)
	if !ok || pathArg == "" {
		return map[string]any{"error": "path is required"}, nil
	}

	fullPath := e.resolvePath(pathArg)

	// Security check
	if !e.isPathAllowed(fullPath) {
		return map[string]any{"error": "path is outside allowed directory"}, nil
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"error": fmt.Sprintf("file not found: %s", pathArg)}, nil
		}
		return map[string]any{"error": err.Error()}, nil
	}

	if info.IsDir() {
		return map[string]any{"error": "path is a directory, use list_directory instead"}, nil
	}

	if info.Size() > e.maxFileSize {
		return map[string]any{
			"error": fmt.Sprintf("file too large: %d bytes (max %d bytes)", info.Size(), e.maxFileSize),
			"path":  fullPath,
			"size":  info.Size(),
		}, nil
	}

	// Check for binary file
	if e.isBinaryFile(fullPath) {
		return map[string]any{
			"error": "file appears to be binary",
			"path":  fullPath,
			"size":  info.Size(),
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

// listDirectory lists contents of a directory
func (e *Executor) listDirectory(args map[string]any) (map[string]any, error) {
	pathArg, _ := args["path"].(string)
	if pathArg == "" {
		pathArg = "."
	}

	fullPath := e.resolvePath(pathArg)

	// Security check
	if !e.isPathAllowed(fullPath) {
		return map[string]any{"error": "path is outside allowed directory"}, nil
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"error": fmt.Sprintf("directory not found: %s", pathArg)}, nil
		}
		return map[string]any{"error": err.Error()}, nil
	}

	if !info.IsDir() {
		return map[string]any{"error": "path is not a directory"}, nil
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}

	var items []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		items = append(items, name)
	}

	return map[string]any{
		"path":  fullPath,
		"items": items,
		"count": len(items),
	}, nil
}

// globSearch finds files matching a glob pattern
func (e *Executor) globSearch(args map[string]any) (map[string]any, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return map[string]any{"error": "pattern is required"}, nil
	}

	// Use doublestar for ** support
	matches, err := doublestar.Glob(os.DirFS(e.workingDir), pattern)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("invalid pattern: %s", err.Error())}, nil
	}

	// Limit results
	truncated := false
	if len(matches) > e.maxResults {
		matches = matches[:e.maxResults]
		truncated = true
	}

	return map[string]any{
		"pattern":   pattern,
		"matches":   matches,
		"count":     len(matches),
		"truncated": truncated,
	}, nil
}

// writeFile writes content to a file
func (e *Executor) writeFile(args map[string]any) (map[string]any, error) {
	pathArg, ok := args["path"].(string)
	if !ok || pathArg == "" {
		return map[string]any{"error": "path is required"}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return map[string]any{"error": "content is required"}, nil
	}

	fullPath := e.resolvePath(pathArg)

	// Security check
	if !e.isPathAllowed(fullPath) {
		return map[string]any{"error": "path is outside allowed directory"}, nil
	}

	// Check if file size would exceed limit
	if int64(len(content)) > e.maxFileSize*10 { // Allow larger writes than reads
		return map[string]any{
			"error": fmt.Sprintf("content too large: %d bytes (max %d bytes)", len(content), e.maxFileSize*10),
		}, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to create directory: %s", err.Error())}, nil
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return map[string]any{"error": err.Error()}, nil
	}

	return map[string]any{
		"path":    fullPath,
		"size":    len(content),
		"success": true,
	}, nil
}

// editFile edits an existing file by replacing a string
func (e *Executor) editFile(args map[string]any) (map[string]any, error) {
	pathArg, ok := args["path"].(string)
	if !ok || pathArg == "" {
		return map[string]any{"error": "path is required"}, nil
	}

	oldString, ok := args["old_string"].(string)
	if !ok {
		return map[string]any{"error": "old_string is required"}, nil
	}

	newString, ok := args["new_string"].(string)
	if !ok {
		return map[string]any{"error": "new_string is required"}, nil
	}

	fullPath := e.resolvePath(pathArg)

	// Security check
	if !e.isPathAllowed(fullPath) {
		return map[string]any{"error": "path is outside allowed directory"}, nil
	}

	// Read the file
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"error": fmt.Sprintf("file not found: %s", pathArg)}, nil
		}
		return map[string]any{"error": err.Error()}, nil
	}

	content := string(contentBytes)

	// Check if old_string exists in the file
	if !strings.Contains(content, oldString) {
		return map[string]any{
			"error": "old_string not found in file",
			"path":  fullPath,
		}, nil
	}

	// Check if old_string is unique
	count := strings.Count(content, oldString)
	if count > 1 {
		return map[string]any{
			"error":       fmt.Sprintf("old_string found %d times in file, must be unique", count),
			"path":        fullPath,
			"occurrences": count,
		}, nil
	}

	// Replace the string
	newContent := strings.Replace(content, oldString, newString, 1)

	// Write the file back
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return map[string]any{"error": err.Error()}, nil
	}

	return map[string]any{
		"path":    fullPath,
		"size":    len(newContent),
		"success": true,
	}, nil
}

// createDirectory creates a directory and any necessary parents
func (e *Executor) createDirectory(args map[string]any) (map[string]any, error) {
	pathArg, ok := args["path"].(string)
	if !ok || pathArg == "" {
		return map[string]any{"error": "path is required"}, nil
	}

	fullPath := e.resolvePath(pathArg)

	// Security check
	if !e.isPathAllowed(fullPath) {
		return map[string]any{"error": "path is outside allowed directory"}, nil
	}

	// Create the directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return map[string]any{"error": err.Error()}, nil
	}

	return map[string]any{
		"path":    fullPath,
		"success": true,
	}, nil
}

// resolvePath resolves a path relative to the working directory
func (e *Executor) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(e.workingDir, path))
}

// isPathAllowed checks if a path is within the allowed directory
func (e *Executor) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if the path is within or equal to the working directory
	rel, err := filepath.Rel(e.workingDir, absPath)
	if err != nil {
		return false
	}

	// If the relative path starts with "..", it's outside the working directory
	return !strings.HasPrefix(rel, "..")
}

// isBinaryFile checks if a file appears to be binary by reading first bytes
func (e *Executor) isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first 512 bytes to check for binary content
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}
