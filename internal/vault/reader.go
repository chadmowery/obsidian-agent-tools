package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Reader struct {
	vaultPath string
}

func NewReader(vaultPath string) *Reader {
	return &Reader{vaultPath: vaultPath}
}

// ReadNote reads a note by filename
func (r *Reader) ReadNote(filename string) (string, error) {
	fullPath := filepath.Join(r.vaultPath, filename)
	if !strings.HasSuffix(fullPath, ".md") {
		fullPath += ".md"
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read note: %w", err)
	}

	return string(content), nil
}

// SearchNotes performs a simple text search across all markdown files
func (r *Reader) SearchNotes(query string) ([]string, error) {
	var results []string

	err := filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process markdown files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			if strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
				relPath, _ := filepath.Rel(r.vaultPath, path)
				results = append(results, relPath)
			}
		}

		return nil
	})

	return results, err
}

// GetDailyNote returns the daily note for a given date
func (r *Reader) GetDailyNote(date string) (string, error) {
	// Parse date or use today
	var targetDate time.Time
	var err error

	if date == "" || date == "today" {
		targetDate = time.Now()
	} else {
		targetDate, err = time.Parse("2006-01-02", date)
		if err != nil {
			return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
		}
	}

	// Common daily note formats
	formats := []string{
		"Rough Notes/%s.md",
		"Daily/%s.md",
		"%s.md",
	}

	dateStr := targetDate.Format("2006-01-02")

	for _, format := range formats {
		filename := fmt.Sprintf(format, dateStr)
		fullPath := filepath.Join(r.vaultPath, filename)

		if _, err := os.Stat(fullPath); err == nil {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}

	return "", fmt.Errorf("daily note not found for %s", dateStr)
}

// ListTags extracts all unique tags from the vault
func (r *Reader) ListTags() ([]string, error) {
	tagSet := make(map[string]bool)

	err := filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			// Extract #tags
			words := strings.Fields(string(content))
			for _, word := range words {
				if strings.HasPrefix(word, "#") {
					tag := strings.TrimPrefix(word, "#")
					tag = strings.Trim(tag, ".,!?;:")
					if tag != "" {
						tagSet[tag] = true
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags, nil
}
