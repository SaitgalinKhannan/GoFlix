package torrent

import (
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

// Service coordinates the torrent client, persistent state, the concurrent-download
// scheduler, and SSE status broadcasting.
type Service struct {
	client        *Client
	state         *StateManager
	hub           *StatusHub
	maxConcurrent int

	// seededStatuses feeds per-torrent stored metadata into the status merge.
	// Held under its own mutex so the broadcaster can run without the client lock.
	storedMu sync.Mutex
}

// NewService creates a new torrent service.
func NewService(client *Client, sm *StateManager, hub *StatusHub, maxConcurrent int) *Service {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Service{
		client:        client,
		state:         sm,
		hub:           hub,
		maxConcurrent: maxConcurrent,
	}
}

// AddMagnet adds a torrent from a magnet link.
func (s *Service) AddMagnet(magnet string) (*TorrentStatus, error) {
	infoHash, err := s.client.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}
	name := s.client.torrentName(infoHash)
	size := s.client.torrentSize(infoHash)
	s.state.Add(&storedTorrent{
		InfoHash: infoHash,
		Name:     name,
		Magnet:   magnet,
		Size:     size,
		AddedAt:  time.Now(),
		Seeding:  true,
	})
	s.publish()
	return s.statusOf(infoHash), nil
}

// AddFile adds a torrent from a .torrent file path.
func (s *Service) AddFile(path, magnet string) (*TorrentStatus, error) {
	infoHash, err := s.client.AddFile(path)
	if err != nil {
		return nil, err
	}
	name := s.client.torrentName(infoHash)
	size := s.client.torrentSize(infoHash)
	s.state.Add(&storedTorrent{
		InfoHash: infoHash,
		Name:     name,
		Magnet:   magnet,
		Size:     size,
		AddedAt:  time.Now(),
		Seeding:  true,
	})
	s.publish()
	return s.statusOf(infoHash), nil
}

// List returns the merged status of all known torrents (active + stored).
func (s *Service) List() []TorrentStatus {
	active := s.client.Statuses()
	activeMap := make(map[string]TorrentStatus, len(active))
	for _, a := range active {
		activeMap[a.InfoHash] = a
	}

	stored := s.state.All()
	out := make([]TorrentStatus, 0, len(stored))
	for _, st := range stored {
		if a, ok := activeMap[st.InfoHash]; ok {
			// merge stored metadata + completion into the live status
			a.Name = orDefault(a.Name, st.Name)
			a.Size = orSize(a.Size, st.Size)
			a.AddedAt = st.AddedAt
			a.CompletedAt = st.CompletedAt
			out = append(out, a)
		} else {
			// not active in client → reflect stored/paused state
			status := StatusPaused
			var progress float64
			var downloaded int64
			if st.CompletedAt != nil {
				status = StatusCompleted
				progress = 1
				downloaded = st.Size
			}
			out = append(out, TorrentStatus{
				ID:          st.InfoHash,
				InfoHash:    st.InfoHash,
				Name:        st.Name,
				Status:      status,
				Progress:    progress,
				Size:        st.Size,
				Downloaded:  downloaded,
				AddedAt:     st.AddedAt,
				CompletedAt: st.CompletedAt,
			})
		}
	}

	// Include active torrents that aren't yet persisted (rare race).
	for _, a := range active {
		if _, ok := activeMap2(out, a.InfoHash); !ok {
			out = append(out, a)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].AddedAt.After(out[j].AddedAt)
	})
	return out
}

// Get returns a single torrent's status, including its files.
func (s *Service) Get(infoHash string) (*TorrentStatus, error) {
	status := s.statusOf(infoHash)
	if status == nil {
		return nil, fmt.Errorf("torrent with infohash %s not found", infoHash)
	}
	if files, err := s.client.Files(infoHash); err == nil {
		status.Files = files
	}
	return status, nil
}

// Delete removes a torrent. If deleteFiles is true, downloaded files are removed from disk.
func (s *Service) Delete(infoHash string, deleteFiles bool) error {
	st, ok := s.state.Get(infoHash)

	// Fully drop the torrent from the client (not just pause).
	s.client.dropTorrent(infoHash)
	if deleteFiles && ok {
		s.client.DeleteFiles(st.Name)
	}
	s.state.Remove(infoHash)
	s.publish()
	return nil
}

// StartSeeding re-activates a paused torrent (keeps its data).
func (s *Service) StartSeeding(infoHash string) error {
	st, ok := s.state.Get(infoHash)
	if !ok {
		return fmt.Errorf("torrent with infohash %s not found", infoHash)
	}
	if st.Magnet == "" {
		return fmt.Errorf("no magnet link stored for torrent %s", infoHash)
	}
	if _, err := s.client.StartSeeding(st.Magnet); err != nil {
		s.state.SetSeeding(infoHash, false)
		return fmt.Errorf("failed to start seeding: %w", err)
	}
	s.state.SetSeeding(infoHash, true)
	s.publish()
	return nil
}

// StopSeeding stops a torrent (drops it from the client, files preserved).
func (s *Service) StopSeeding(infoHash string) error {
	if !s.state.SetSeeding(infoHash, false) {
		return fmt.Errorf("torrent with infohash %s not found", infoHash)
	}
	if err := s.client.StopSeeding(infoHash); err != nil {
		return err
	}
	s.publish()
	return nil
}

// StartBackgroundWorkers runs the status broadcaster, the concurrent-download
// scheduler, and the completion monitor. All stop on sigChan.
func (s *Service) StartBackgroundWorkers(sigChan chan os.Signal) {
	// Re-add persisted, user-seeding torrents on startup.
	s.resumeOnStartup()

	// Broadcaster: builds statuses every second, marks completions, pushes to SSE hub.
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-sigChan:
				log.Println("Stopping torrent monitor...")
				return
			}
		}
	}()

	// Scheduler: enforces the concurrent active-download limit.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.enforceLimit()
			case <-sigChan:
				log.Println("Stopping scheduler...")
				return
			}
		}
	}()
}

// resumeOnStartup re-adds every stored torrent the user wants to seed.
func (s *Service) resumeOnStartup() {
	for _, st := range s.state.All() {
		if !st.Seeding || st.Magnet == "" {
			continue
		}
		go func(st storedTorrent) {
			log.Printf("Resuming torrent on startup: %s", st.Name)
			if _, err := s.client.StartSeeding(st.Magnet); err != nil {
				log.Printf("Failed to resume torrent %s: %v", st.Name, err)
			}
		}(st)
	}
}

// tick samples speeds (advances the EMA window), marks newly completed
// torrents, and broadcasts. Called once per second.
func (s *Service) tick() {
	s.client.sampleSpeeds()
	for _, status := range s.client.Statuses() {
		if status.Status == StatusSeeding || status.Status == StatusCompleted {
			s.state.MarkCompleted(status.InfoHash)
		}
	}
	s.publish()
}

// enforceLimit pauses low-priority downloads beyond maxConcurrent and resumes
// paused ones when slots free up. Paused torrents stay in the client so their
// piece-completion is preserved for instant resume.
func (s *Service) enforceLimit() {
	active := s.client.Statuses()

	// Count torrents currently downloading (not paused, not completed).
	var downloading []TorrentStatus
	var pausedNotDone []TorrentStatus
	for _, a := range active {
		switch a.Status {
		case StatusDownloading, StatusMetadata:
			downloading = append(downloading, a)
		case StatusPaused:
			if a.Progress < 1 {
				pausedNotDone = append(pausedNotDone, a)
			}
		}
	}

	// If under the limit, resume paused torrents to fill slots.
	sort.Slice(pausedNotDone, func(i, j int) bool {
		return pausedNotDone[i].Progress > pausedNotDone[j].Progress // most-progressed first
	})
	for _, p := range pausedNotDone {
		if len(downloading) >= s.maxConcurrent {
			break
		}
		// Only resume torrents the scheduler itself paused; skip user-paused ones.
		if st, ok := s.state.Get(p.InfoHash); ok && !st.Seeding {
			continue
		}
		log.Printf("[scheduler] resuming torrent: %s", p.Name)
		if st, ok := s.state.Get(p.InfoHash); ok {
			if _, err := s.client.StartSeeding(st.Magnet); err == nil {
				downloading = append(downloading, p)
			}
		}
	}

	if len(downloading) <= s.maxConcurrent {
		return
	}
	// Pause the least-progressed downloads beyond the limit.
	sort.Slice(downloading, func(i, j int) bool {
		return downloading[i].Progress < downloading[j].Progress
	})
	for _, d := range downloading[s.maxConcurrent:] {
		log.Printf("[scheduler] pausing torrent to respect concurrent limit: %s", d.Name)
		if err := s.client.StopSeeding(d.InfoHash); err != nil {
			log.Printf("[scheduler] error stopping %s: %v", d.InfoHash, err)
		}
	}
}

// publish broadcasts current statuses to all SSE subscribers.
func (s *Service) publish() {
	if s.hub == nil {
		return
	}
	s.hub.Broadcast(s.List())
}

// Subscribe registers an SSE client and returns its receive channel and an
// initial snapshot. Call Unsubscribe when the client disconnects.
func (s *Service) Subscribe() (chan []byte, []byte) {
	if s.hub == nil {
		ch := make(chan []byte)
		return ch, nil
	}
	return s.hub.Subscribe()
}

// Unsubscribe removes an SSE client.
func (s *Service) Unsubscribe(ch chan []byte) {
	if s.hub == nil {
		return
	}
	s.hub.Unsubscribe(ch)
}

// statusOf returns the merged status for a single torrent, or nil if unknown.
func (s *Service) statusOf(infoHash string) *TorrentStatus {
	for _, status := range s.List() {
		if status.InfoHash == infoHash {
			s := status
			return &s
		}
	}
	return nil
}

// orDefault returns v if non-empty, else fallback.
func orDefault(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// orSize returns v if positive, else fallback.
func orSize(v, fallback int64) int64 {
	if v > 0 {
		return v
	}
	return fallback
}

// activeMap2 checks whether infoHash exists in the slice.
func activeMap2(statuses []TorrentStatus, infoHash string) (TorrentStatus, bool) {
	for _, s := range statuses {
		if s.InfoHash == infoHash {
			return s, true
		}
	}
	return TorrentStatus{}, false
}
