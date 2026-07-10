package torrent

import (
	"GoFlix/internal/pkg/filehelpers"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateManager persists known torrents (infohash, name, magnet, completion,
// seeding preference) to a JSON file and re-broadcasts completion events.
type StateManager struct {
	mu        sync.RWMutex
	states    map[string]*storedTorrent
	stateFile string

	saveChan chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	// completion event channel, consumed by the event handler / Jellyfin hook
	completedChan chan *storedTorrent
}

// NewStateManager creates a state manager and loads any persisted torrents.
func NewStateManager(stateFile string) *StateManager {
	sm := &StateManager{
		states:        make(map[string]*storedTorrent),
		stateFile:     stateFile,
		saveChan:      make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		completedChan: make(chan *storedTorrent, 100),
	}
	if err := sm.load(); err != nil {
		log.Printf("Warning: failed to load states: %v", err)
	}
	sm.startBackground()
	return sm
}

// InitStateFile creates an empty state file if none exists.
func InitStateFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
			return mkErr
		}
		file, createErr := os.Create(path)
		if createErr != nil {
			return createErr
		}
		defer filehelpers.CloseFile(file)
		if encErr := json.NewEncoder(file).Encode(map[string]any{}); encErr != nil {
			return encErr
		}
		log.Println("Created state file:", path)
	}
	return nil
}

func (sm *StateManager) load() error {
	file, err := os.Open(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer filehelpers.CloseFile(file)

	if err := json.NewDecoder(file).Decode(&sm.states); err != nil {
		return fmt.Errorf("failed to decode states: %w", err)
	}
	log.Printf("Loaded %d torrent states from file", len(sm.states))
	return nil
}

func (sm *StateManager) save() error {
	sm.mu.RLock()
	copy := make(map[string]*storedTorrent, len(sm.states))
	for k, v := range sm.states {
		val := *v
		copy[k] = &val
	}
	sm.mu.RUnlock()

	tmp := sm.stateFile + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(copy); err != nil {
		filehelpers.CloseFile(file)
		filehelpers.OsRemove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		filehelpers.OsRemove(tmp)
		return err
	}
	return os.Rename(tmp, sm.stateFile)
}

func (sm *StateManager) scheduleSave() {
	select {
	case sm.saveChan <- struct{}{}:
	default:
	}
}

func (sm *StateManager) startBackground() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sm.stopChan:
				return
			case <-ticker.C:
				if err := sm.save(); err != nil {
					log.Printf("Error saving states: %v", err)
				}
			case <-sm.saveChan:
				if err := sm.save(); err != nil {
					log.Printf("Error saving states: %v", err)
				}
			}
		}
	}()
}

// Add records a newly added torrent.
func (sm *StateManager) Add(t *storedTorrent) {
	sm.mu.Lock()
	if existing, ok := sm.states[t.InfoHash]; ok {
		// Preserve addedAt / completedAt if already known.
		t.AddedAt = existing.AddedAt
		t.CompletedAt = existing.CompletedAt
	}
	sm.states[t.InfoHash] = t
	sm.mu.Unlock()
	sm.scheduleSave()
}

// Get returns a copy of a stored torrent.
func (sm *StateManager) Get(infoHash string) (*storedTorrent, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	t, ok := sm.states[infoHash]
	if !ok {
		return nil, false
	}
	copy := *t
	return &copy, true
}

// All returns copies of all stored torrents.
func (sm *StateManager) All() []storedTorrent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]storedTorrent, 0, len(sm.states))
	for _, t := range sm.states {
		out = append(out, *t)
	}
	return out
}

// SetSeeding updates the user's seeding preference for a torrent.
func (sm *StateManager) SetSeeding(infoHash string, seeding bool) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	t, ok := sm.states[infoHash]
	if !ok {
		return false
	}
	t.Seeding = seeding
	sm.scheduleSave()
	return true
}

// MarkCompleted records the completion time and emits a completion event
// (used to trigger the Jellyfin library refresh). Returns true if this call
// is the one that marked completion (i.e. it was newly completed).
func (sm *StateManager) MarkCompleted(infoHash string) bool {
	sm.mu.Lock()
	t, ok := sm.states[infoHash]
	newlyCompleted := false
	if ok {
		if t.CompletedAt == nil {
			now := time.Now()
			t.CompletedAt = &now
			newlyCompleted = true
			sm.scheduleSave()
		}
	}
	sm.mu.Unlock()

	if newlyCompleted && t != nil {
		copy := *t
		select {
		case sm.completedChan <- &copy:
		default:
			log.Println("Completed channel full, dropping event")
		}
	}
	return newlyCompleted
}

// Remove deletes a torrent from state.
func (sm *StateManager) Remove(infoHash string) {
	sm.mu.Lock()
	delete(sm.states, infoHash)
	sm.mu.Unlock()
	sm.scheduleSave()
}

// CompletedChan returns the channel that fires when a torrent finishes downloading.
func (sm *StateManager) CompletedChan() <-chan *storedTorrent {
	return sm.completedChan
}

// Stop halts background processes and flushes state to disk.
func (sm *StateManager) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()
	if err := sm.save(); err != nil {
		log.Printf("Error during final save: %v", err)
	}
	close(sm.completedChan)
}
