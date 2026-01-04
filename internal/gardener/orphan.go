package gardener

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// wikilinkRegex matches [[wikilinks]] including aliased links [[target|alias]]
var wikilinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// LinkGraph represents the link structure of the vault
type LinkGraph struct {
	// Outgoing maps a note to all notes it links to
	Outgoing map[string][]string

	// Incoming maps a note to all notes that link to it
	Incoming map[string][]string

	// AllNotes is a list of all notes in the vault
	AllNotes []string
}

// OrphanFinder identifies orphan notes in the vault
type OrphanFinder struct {
	vaultPath string
}

// NewOrphanFinder creates a new OrphanFinder
func NewOrphanFinder(vaultPath string) *OrphanFinder {
	return &OrphanFinder{vaultPath: vaultPath}
}

// BuildLinkGraph scans the vault and builds a link graph
func (o *OrphanFinder) BuildLinkGraph() (*LinkGraph, error) {
	graph := &LinkGraph{
		Outgoing: make(map[string][]string),
		Incoming: make(map[string][]string),
	}

	// First pass: collect all notes
	err := filepath.Walk(o.vaultPath, func(path string, info os.FileInfo, err error) error {
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
			relPath, _ := filepath.Rel(o.vaultPath, path)
			graph.AllNotes = append(graph.AllNotes, relPath)
			graph.Outgoing[relPath] = []string{}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: extract links
	for _, notePath := range graph.AllNotes {
		fullPath := filepath.Join(o.vaultPath, notePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Find all wikilinks
		matches := wikilinkRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			target := match[1]
			// Normalize the target (add .md if needed, resolve relative paths)
			targetPath := normalizeLink(target, notePath)

			// Check if target exists in our graph
			if _, exists := graph.Outgoing[targetPath]; exists {
				graph.Outgoing[notePath] = append(graph.Outgoing[notePath], targetPath)

				if graph.Incoming[targetPath] == nil {
					graph.Incoming[targetPath] = []string{}
				}
				graph.Incoming[targetPath] = append(graph.Incoming[targetPath], notePath)
			}
		}
	}

	return graph, nil
}

// normalizeLink converts a wikilink target to a file path
func normalizeLink(target, sourcePath string) string {
	// Remove any leading/trailing whitespace
	target = strings.TrimSpace(target)

	// Add .md extension if not present
	if !strings.HasSuffix(target, ".md") {
		target = target + ".md"
	}

	return target
}

// FindOrphans returns notes with no incoming or outgoing links
func (o *OrphanFinder) FindOrphans() ([]string, error) {
	graph, err := o.BuildLinkGraph()
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, note := range graph.AllNotes {
		outgoing := len(graph.Outgoing[note])
		incoming := len(graph.Incoming[note])

		if outgoing == 0 && incoming == 0 {
			orphans = append(orphans, note)
		}
	}

	return orphans, nil
}

// FindDeadEnds returns notes with incoming links but no outgoing links
func (o *OrphanFinder) FindDeadEnds() ([]string, error) {
	graph, err := o.BuildLinkGraph()
	if err != nil {
		return nil, err
	}

	var deadEnds []string
	for _, note := range graph.AllNotes {
		outgoing := len(graph.Outgoing[note])
		incoming := len(graph.Incoming[note])

		if outgoing == 0 && incoming > 0 {
			deadEnds = append(deadEnds, note)
		}
	}

	return deadEnds, nil
}

// FindIslands returns notes that only link to each other (isolated clusters)
// This is a simplified version - just finds notes with no external connections
func (o *OrphanFinder) FindIslands() ([]string, error) {
	graph, err := o.BuildLinkGraph()
	if err != nil {
		return nil, err
	}

	var islands []string
	for _, note := range graph.AllNotes {
		outgoing := len(graph.Outgoing[note])
		incoming := len(graph.Incoming[note])

		// Has links but is isolated from main graph
		if outgoing > 0 && incoming == 0 {
			islands = append(islands, note)
		}
	}

	return islands, nil
}

// GetLinkStats returns statistics about the link graph
func (o *OrphanFinder) GetLinkStats() (map[string]int, error) {
	graph, err := o.BuildLinkGraph()
	if err != nil {
		return nil, err
	}

	orphanCount := 0
	deadEndCount := 0
	wellLinkedCount := 0

	for _, note := range graph.AllNotes {
		outgoing := len(graph.Outgoing[note])
		incoming := len(graph.Incoming[note])

		if outgoing == 0 && incoming == 0 {
			orphanCount++
		} else if outgoing == 0 {
			deadEndCount++
		} else if incoming > 0 {
			wellLinkedCount++
		}
	}

	return map[string]int{
		"total_notes": len(graph.AllNotes),
		"orphans":     orphanCount,
		"dead_ends":   deadEndCount,
		"well_linked": wellLinkedCount,
	}, nil
}
