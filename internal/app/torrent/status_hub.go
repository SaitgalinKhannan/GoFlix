package torrent

import (
	"encoding/json"
	"sync"
)

// StatusHub fans out torrent status snapshots to all subscribed SSE clients.
type StatusHub struct {
	mu        sync.RWMutex
	clients   map[chan []byte]struct{}
	lastBatch []byte // most recent snapshot, sent to new subscribers
}

// NewStatusHub creates a new status hub.
func NewStatusHub() *StatusHub {
	return &StatusHub{clients: make(map[chan []byte]struct{})}
}

// Subscribe registers a new SSE client and returns its receive channel plus
// an initial snapshot. Unsubscribe must be called when the client disconnects.
func (h *StatusHub) Subscribe() (chan []byte, []byte) {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	snapshot := append([]byte(nil), h.lastBatch...)
	h.mu.Unlock()
	return ch, snapshot
}

// Unsubscribe removes a client. The channel is left open (GC handles it) to
// avoid a race where Broadcast sends on a closed channel.
func (h *StatusHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast sends a status snapshot to every subscriber. Non-blocking: slow
// clients are skipped to avoid stalling the broadcaster.
func (h *StatusHub) Broadcast(statuses []TorrentStatus) {
	data, err := json.Marshal(statuses)
	if err != nil {
		return
	}
	h.mu.RLock()
	h.lastBatch = append([]byte(nil), data...)
	clients := make([]chan []byte, 0, len(h.clients))
	for ch := range h.clients {
		clients = append(clients, ch)
	}
	h.mu.RUnlock()

	for _, ch := range clients {
		select {
		case ch <- data:
		default: // drop for slow clients
		}
	}
}
