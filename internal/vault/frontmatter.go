package vault

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents the YAML frontmatter of a markdown file
type Frontmatter map[string]interface{}

// frontmatterRegex matches YAML frontmatter at the start of a file
var frontmatterRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?`)

// ParseFrontmatter extracts frontmatter and content from markdown
func ParseFrontmatter(content string) (Frontmatter, string, error) {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		// No frontmatter found
		return nil, content, nil
	}

	fm := make(Frontmatter)
	if err := yaml.Unmarshal([]byte(matches[1]), &fm); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Remove frontmatter from content
	body := frontmatterRegex.ReplaceAllString(content, "")
	return fm, body, nil
}

// SerializeFrontmatter converts frontmatter map to YAML string
func SerializeFrontmatter(fm Frontmatter) (string, error) {
	if len(fm) == 0 {
		return "", nil
	}

	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("failed to serialize frontmatter: %w", err)
	}

	return fmt.Sprintf("---\n%s---\n", string(data)), nil
}

// CombineFrontmatterAndContent creates a complete markdown file
func CombineFrontmatterAndContent(fm Frontmatter, content string) (string, error) {
	fmStr, err := SerializeFrontmatter(fm)
	if err != nil {
		return "", err
	}

	// Ensure content starts on a new line if frontmatter exists
	if fmStr != "" && !strings.HasPrefix(content, "\n") {
		content = "\n" + content
	}

	return fmStr + content, nil
}

// MergeFrontmatter merges new values into existing frontmatter
func MergeFrontmatter(existing, updates Frontmatter) Frontmatter {
	if existing == nil {
		existing = make(Frontmatter)
	}

	for k, v := range updates {
		existing[k] = v
	}

	return existing
}
