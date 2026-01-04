package watcher

import (
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the vault for changes
type Watcher struct {
	watcher *fsnotify.Watcher
	done    chan bool
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
	}, nil
}

// Start begins watching the specified directory recursively
// Note: fsnotify doesn't support recursive watching out of the box on all OSs,
// so for a robust implementation we'd need to walk the tree.
// For now, we'll implement a simple top-level watch and todo for recursion.
func (w *Watcher) Start(vaultPath string) {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				// Ignore hidden files and .obsidian config
				if strings.Contains(event.Name, "/.") {
					continue
				}

				log.Printf("Event: %v", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Modified file:", event.Name)
					// TODO: Trigger re-index or callback
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("Created file:", event.Name)
					// If directory, add to watch list (naive recursion)
				}

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("Verifier error:", err)
			}
		}
	}()

	err := w.watcher.Add(vaultPath)
	if err != nil {
		log.Printf("Error watching vault root: %v", err)
	}

	// TODO: Walk directory and add subfolders
}

// Close stops the watcher
func (w *Watcher) Close() {
	w.watcher.Close()
	close(w.done)
}
