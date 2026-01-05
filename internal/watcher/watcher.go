package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileChangeCallback is called when a file is created, modified, or deleted
type FileChangeCallback func(path string, op FileOp)

// FileOp represents the type of file operation
type FileOp int

const (
	OpCreate FileOp = iota
	OpModify
	OpDelete
)

// Watcher monitors the vault for changes
type Watcher struct {
	watcher   *fsnotify.Watcher
	done      chan bool
	callback  FileChangeCallback
	debouncer *debouncer
	mu        sync.Mutex
}

// debouncer handles debouncing of file events
type debouncer struct {
	events map[string]*debouncedEvent
	mu     sync.Mutex
	delay  time.Duration
}

type debouncedEvent struct {
	path  string
	op    FileOp
	timer *time.Timer
}

// NewWatcher creates a new Watcher instance
func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher: w,
		done:    make(chan bool),
		debouncer: &debouncer{
			events: make(map[string]*debouncedEvent),
			delay:  300 * time.Millisecond,
		},
	}, nil
}

// SetCallback sets the callback function for file changes
func (w *Watcher) SetCallback(cb FileChangeCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callback = cb
}

// Start begins watching the specified directory recursively
func (w *Watcher) Start(vaultPath string) error {
	// Walk directory tree and add all subdirectories to watch
	err := filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(filepath.Base(path), ".") && path != vaultPath {
			return filepath.SkipDir
		}

		// Add directory to watch
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				log.Printf("Warning: failed to watch %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Start event processing goroutine
	go w.processEvents()

	log.Printf("📁 Watching vault for changes: %s", vaultPath)
	return nil
}

// processEvents handles file system events
func (w *Watcher) processEvents() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Ignore hidden files and directories
			if strings.Contains(event.Name, "/.") {
				continue
			}

			// Only process markdown files
			if !strings.HasSuffix(event.Name, ".md") {
				// Check if it's a directory creation (need to add to watch)
				if event.Op&fsnotify.Create == fsnotify.Create {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						w.watcher.Add(event.Name)
						log.Printf("📂 Added new directory to watch: %s", event.Name)
					}
				}
				continue
			}

			// Determine operation type
			var op FileOp
			if event.Op&fsnotify.Create == fsnotify.Create {
				op = OpCreate
			} else if event.Op&fsnotify.Write == fsnotify.Write {
				op = OpModify
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				op = OpDelete
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				op = OpDelete // Treat rename as delete (new file will trigger create)
			} else {
				continue
			}

			// Debounce the event
			w.debounceEvent(event.Name, op)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("⚠️  Watcher error: %v", err)

		case <-w.done:
			return
		}
	}
}

// debounceEvent debounces file events to avoid processing rapid successive changes
func (w *Watcher) debounceEvent(path string, op FileOp) {
	w.debouncer.mu.Lock()
	defer w.debouncer.mu.Unlock()

	// Cancel existing timer if present
	if existing, ok := w.debouncer.events[path]; ok {
		existing.timer.Stop()
	}

	// Create new debounced event
	event := &debouncedEvent{
		path: path,
		op:   op,
	}

	event.timer = time.AfterFunc(w.debouncer.delay, func() {
		w.debouncer.mu.Lock()
		delete(w.debouncer.events, path)
		w.debouncer.mu.Unlock()

		// Execute callback
		w.mu.Lock()
		cb := w.callback
		w.mu.Unlock()

		if cb != nil {
			cb(path, op)
		}
	})

	w.debouncer.events[path] = event
}

// Close stops the watcher
func (w *Watcher) Close() {
	close(w.done)

	// Cancel all pending debounced events
	w.debouncer.mu.Lock()
	for _, event := range w.debouncer.events {
		event.timer.Stop()
	}
	w.debouncer.mu.Unlock()

	w.watcher.Close()
}
