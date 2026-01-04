package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Writer provides write operations for the Obsidian vault
type Writer struct {
	vaultPath string
}

// NewWriter creates a new Writer instance
func NewWriter(vaultPath string) *Writer {
	return &Writer{vaultPath: vaultPath}
}

// AppendToDailyNote appends a timestamped entry to today's daily note
// Creates the note if it doesn't exist
func (w *Writer) AppendToDailyNote(text string) error {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timestamp := now.Format("15:04")

	// Try common daily note locations
	dailyNotePaths := []string{
		filepath.Join("Rough Notes", dateStr+".md"),
		filepath.Join("Daily", dateStr+".md"),
		dateStr + ".md",
	}

	var targetPath string
	var existingContent string

	// Find existing daily note or use first path
	for _, p := range dailyNotePaths {
		fullPath := filepath.Join(w.vaultPath, p)
		if content, err := os.ReadFile(fullPath); err == nil {
			targetPath = fullPath
			existingContent = string(content)
			break
		}
	}

	// If no existing note found, create in Rough Notes
	if targetPath == "" {
		targetPath = filepath.Join(w.vaultPath, dailyNotePaths[0])
		// Create the directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create daily note directory: %w", err)
		}
		// Initialize with frontmatter
		existingContent = fmt.Sprintf("---\ndate: %s\ntags:\n  - daily-note\n---\n\n# %s\n\n", dateStr, dateStr)
	}

	// Append the timestamped entry
	entry := fmt.Sprintf("\n- **%s** %s\n", timestamp, text)
	newContent := existingContent + entry

	if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write daily note: %w", err)
	}

	return nil
}

// CreateNote creates a new note with optional frontmatter
func (w *Writer) CreateNote(path, content string, frontmatter Frontmatter) error {
	// Ensure .md extension
	if !strings.HasSuffix(path, ".md") {
		path = path + ".md"
	}

	fullPath := filepath.Join(w.vaultPath, path)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("note already exists: %s", path)
	}

	// Create directory structure if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Add creation timestamp to frontmatter
	if frontmatter == nil {
		frontmatter = make(Frontmatter)
	}
	if _, exists := frontmatter["created"]; !exists {
		frontmatter["created"] = time.Now().Format(time.RFC3339)
	}

	// Combine frontmatter and content
	fileContent, err := CombineFrontmatterAndContent(frontmatter, content)
	if err != nil {
		return fmt.Errorf("failed to create note content: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("failed to write note: %w", err)
	}

	return nil
}

// UpdateFrontmatter updates a specific key in a note's frontmatter
func (w *Writer) UpdateFrontmatter(path, key string, value interface{}) error {
	// Ensure .md extension
	if !strings.HasSuffix(path, ".md") {
		path = path + ".md"
	}

	fullPath := filepath.Join(w.vaultPath, path)

	// Read existing content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read note: %w", err)
	}

	// Parse existing frontmatter
	fm, body, err := ParseFrontmatter(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Initialize frontmatter if needed
	if fm == nil {
		fm = make(Frontmatter)
	}

	// Update the key
	fm[key] = value

	// Rebuild the file
	newContent, err := CombineFrontmatterAndContent(fm, body)
	if err != nil {
		return fmt.Errorf("failed to rebuild note: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write note: %w", err)
	}

	return nil
}

// LinkNotes appends a wikilink from source to target
func (w *Writer) LinkNotes(source, target string) error {
	// Ensure .md extension for source
	if !strings.HasSuffix(source, ".md") {
		source = source + ".md"
	}

	sourcePath := filepath.Join(w.vaultPath, source)

	// Read existing content
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source note: %w", err)
	}

	// Clean target for wikilink (remove .md extension and path)
	linkTarget := strings.TrimSuffix(filepath.Base(target), ".md")
	wikilink := fmt.Sprintf("[[%s]]", linkTarget)

	// Check if link already exists
	if strings.Contains(string(content), wikilink) {
		return nil // Link already exists, no-op
	}

	// Append the link
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("\n## Related\n- %s\n", wikilink)

	if err := os.WriteFile(sourcePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write source note: %w", err)
	}

	return nil
}
