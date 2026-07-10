package torrent

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"
)

// EventHandler reacts to torrent lifecycle events.
// Currently it only fires the Jellyfin library-refresh hook on completion.
type EventHandler struct {
	sm            *StateManager
	jellyfinURL   string
	jellyfinToken string
}

// NewEventHandler creates a new event handler. Empty jellyfinURL/token disables the hook.
func NewEventHandler(sm *StateManager, jellyfinURL, jellyfinToken string) *EventHandler {
	return &EventHandler{
		sm:            sm,
		jellyfinURL:   jellyfinURL,
		jellyfinToken: jellyfinToken,
	}
}

// Start consumes completion events and triggers the Jellyfin hook.
// Returns when the state manager's CompletedChan is closed (on shutdown).
func (eh *EventHandler) Start() {
	go func() {
		for t := range eh.sm.CompletedChan() {
			log.Printf("Torrent download completed: %s", t.Name)
			eh.refreshJellyfin()
		}
	}()
}

func (eh *EventHandler) refreshJellyfin() {
	if eh.jellyfinURL == "" || eh.jellyfinToken == "" {
		return // Jellyfin integration disabled (e.g. local dev).
	}
	url := eh.jellyfinURL + "/Library/Refresh"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		log.Printf("[jellyfin] failed to build request: %v", err)
		return
	}
	req.Header.Set("X-Emby-Token", eh.jellyfinToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[jellyfin] refresh failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[jellyfin] refresh returned %s", resp.Status)
	} else {
		log.Printf("[jellyfin] library refreshed")
	}
}
