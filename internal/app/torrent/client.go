package torrent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

var _ io.Closer = (*Client)(nil)

// Client wraps anacrolix/torrent client and tracks per-torrent byte counters
// so download/upload speeds can be derived between ticks.
type Client struct {
	tClient         *torrent.Client
	baseDir         string
	pieceCompletion storage.PieceCompletion

	mu      sync.Mutex
	samples map[string]*speedTracker // keyed by infoHash
	paused  map[string]bool          // infoHash → paused (DisallowData* called)
}

// speedTracker smooths download/upload speed using exponential moving average,
// so the displayed value doesn't jump wildly between ticks (torrents download
// in bursts as pieces complete).
type speedTracker struct {
	lastDownloaded int64
	lastUploaded   int64
	lastAt         time.Time
	smoothedDL     float64
	smoothedUL     float64
	hasLast        bool
}

// alpha controls smoothing: smaller = smoother/slower to react. 0.15 means
// each new sample contributes 15% to the displayed value.
const alpha = 0.15

func newSpeedTracker() *speedTracker {
	return &speedTracker{}
}

// record takes an absolute byte count and returns the smoothed speed (bytes/s).
func (st *speedTracker) record(dl, ul int64, at time.Time) (dlSpeed, ulSpeed int64) {
	if !st.hasLast {
		st.lastDownloaded = dl
		st.lastUploaded = ul
		st.lastAt = at
		st.hasLast = true
		return 0, 0
	}
	elapsed := at.Sub(st.lastAt).Seconds()
	if elapsed <= 0 {
		return int64(st.smoothedDL), int64(st.smoothedUL)
	}

	// Instantaneous speed for this tick.
	instDL := float64(dl-st.lastDownloaded) / elapsed
	instUL := float64(ul-st.lastUploaded) / elapsed
	if instDL < 0 {
		instDL = 0
	}
	if instUL < 0 {
		instUL = 0
	}

	// Exponential moving average.
	st.smoothedDL = alpha*instDL + (1-alpha)*st.smoothedDL
	st.smoothedUL = alpha*instUL + (1-alpha)*st.smoothedUL

	st.lastDownloaded = dl
	st.lastUploaded = ul
	st.lastAt = at

	return int64(st.smoothedDL), int64(st.smoothedUL)
}

// current returns the last smoothed speeds without recording a new sample.
func (st *speedTracker) current() (dlSpeed, ulSpeed int64) {
	return int64(st.smoothedDL), int64(st.smoothedUL)
}

// NewClient creates a new torrent client downloading into clientBaseDir.
func NewClient(clientBaseDir, pieceCompletionDir string) (*Client, error) {
	if err := os.MkdirAll(clientBaseDir, 0o700); err != nil {
		return nil, err
	}

	// Use an in-memory piece completion (MapPieceCompletion) instead of the
	// default bolt DB. The bolt DB uses NoSync=true which loses piece state on
	// restart, causing torrents to appear to restart from zero. With an in-memory
	// store, anacrolix re-hashes existing files on add — correctly detecting
	// already-downloaded pieces. This makes resume reliable across restarts.
	pieceCompletion := storage.NewMapPieceCompletion()

	opts := storage.NewFileClientOpts{
		ClientBaseDir:   clientBaseDir,
		PieceCompletion: pieceCompletion,
	}

	config := torrent.NewDefaultClientConfig()
	config.DefaultStorage = storage.NewFileOpts(opts)

	tClient, err := torrent.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		tClient:         tClient,
		baseDir:         clientBaseDir,
		pieceCompletion: pieceCompletion,
		samples:         make(map[string]*speedTracker),
		paused:          make(map[string]bool),
	}, nil
}

// BaseDir returns the directory the client downloads into.
func (c *Client) BaseDir() string {
	return c.baseDir
}

// parseHash safely converts a hex infohash into a metainfo.Hash, returning an
// error instead of panicking on malformed input.
func parseHash(infoHash string) (metainfo.Hash, error) {
	var hash metainfo.Hash
	if err := hash.FromHexString(infoHash); err != nil {
		return metainfo.Hash{}, fmt.Errorf("invalid infohash %q: %w", infoHash, err)
	}
	return hash, nil
}

// HasTorrent reports whether a torrent with the given infoHash is active.
func (c *Client) HasTorrent(infoHash string) bool {
	hash, err := parseHash(infoHash)
	if err != nil {
		return false
	}
	_, ok := c.tClient.Torrent(hash)
	return ok
}

// AddMagnet adds a torrent from a magnet link and starts downloading.
func (c *Client) AddMagnet(magnet string) (infoHash string, err error) {
	t, err := c.tClient.AddMagnet(magnet)
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", fmt.Errorf("failed to add torrent: %s", magnet)
	}

	select {
	case <-t.GotInfo():
	case <-time.After(60 * time.Second):
		return "", errors.New("timed out waiting for torrent metadata")
	}
	t.DownloadAll()

	return t.InfoHash().String(), nil
}

// AddFile adds a torrent from a .torrent file path and starts downloading.
func (c *Client) AddFile(path string) (infoHash string, err error) {
	mi, err := metainfo.LoadFromFile(path)
	if err != nil {
		return "", err
	}
	t, err := c.tClient.AddTorrent(mi)
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", fmt.Errorf("failed to add torrent from file: %s", path)
	}

	t.DownloadAll()
	return t.InfoHash().String(), nil
}

// StartSeeding re-enables download+upload for a torrent that's still in the
// client (paused via Stop). If the torrent was dropped (e.g. after a restart),
// it's re-added by magnet — in that case anacrolix re-checks existing data.
func (c *Client) StartSeeding(magnet string) (infoHash string, err error) {
	// Re-adding a torrent that's already in the client is a no-op in anacrolix,
	// and returns the existing *Torrent. We then just re-enable data flow.
	t, err := c.tClient.AddMagnet(magnet)
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", fmt.Errorf("failed to add torrent: %s", magnet)
	}
	select {
	case <-t.GotInfo():
	case <-time.After(60 * time.Second):
		return "", errors.New("timed out waiting for torrent metadata")
	}
	t.AllowDataDownload()
	t.AllowDataUpload()
	t.DownloadAll()
	c.mu.Lock()
	delete(c.paused, t.InfoHash().String())
	c.mu.Unlock()
	return t.InfoHash().String(), nil
}

// StopSeeding pauses a torrent's download and upload WITHOUT dropping it from
// the client, so piece-completion state is preserved and resume is instant.
func (c *Client) StopSeeding(infoHash string) error {
	hash, err := parseHash(infoHash)
	if err != nil {
		return err
	}
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return nil // not in client — nothing to pause
	}
	t.DisallowDataDownload()
	t.DisallowDataUpload()
	c.mu.Lock()
	c.paused[infoHash] = true
	c.mu.Unlock()
	return nil
}

// dropTorrent fully removes a torrent from the client. Used only on Delete.
func (c *Client) dropTorrent(infoHash string) {
	hash, err := parseHash(infoHash)
	if err != nil {
		return
	}
	if t, ok := c.tClient.Torrent(hash); ok {
		t.Drop()
	}
	c.mu.Lock()
	delete(c.paused, infoHash)
	c.mu.Unlock()
}

// isPaused reports whether a torrent was paused via StopSeeding.
// MUST be called with c.mu already held (Statuses uses it inside its lock).
func (c *Client) isPaused(infoHash string) bool {
	return c.paused[infoHash]
}

// DeleteFiles removes the torrent's downloaded data from disk.
func (c *Client) DeleteFiles(name string) {
	if name == "" {
		return
	}
	// Torrents download into baseDir/<name> (or baseDir/<file> for single-file torrents).
	target := filepath.Join(c.baseDir, name)
	if !isWithinBase(c.baseDir, target) {
		log.Printf("[client] refusing to delete path outside base dir: %s", target)
		return
	}
	if err := os.RemoveAll(target); err != nil {
		log.Printf("[client] error removing %s: %v", target, err)
	}
}

// sampleSpeeds records a new speed sample for every active torrent.
// Call this ONCE per second from the broadcaster tick so the EMA window
// advances at a steady rate. Statuses() only reads the smoothed values.
func (c *Client) sampleSpeeds() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, t := range c.tClient.Torrents() {
		infoHash := t.InfoHash().String()
		tracker, ok := c.samples[infoHash]
		if !ok {
			tracker = newSpeedTracker()
			c.samples[infoHash] = tracker
		}
		uploaded := t.Stats().BytesWrittenData
		tracker.record(t.BytesCompleted(), uploaded.Int64(), now)
	}
}

// Statuses builds the current TorrentStatus for every active torrent.
// It only reads the last smoothed speeds (no sampling) so it can be called
// any number of times without distorting the values.
func (c *Client) Statuses() []TorrentStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out []TorrentStatus
	for _, t := range c.tClient.Torrents() {
		info := t.Info()
		infoHash := t.InfoHash().String()
		name := t.Name()

		// info may be nil if metadata isn't available yet.
		var size int64
		if info != nil {
			size = info.TotalLength()
		}

		completed := t.BytesCompleted()
		downloaded := completed

		// Read the last smoothed speed (already updated by sampleSpeeds in tick).
		var dlSpeed, ulSpeed int64
		if tracker, ok := c.samples[infoHash]; ok {
			dlSpeed, ulSpeed = tracker.current()
		}

		var progress float64
		if size > 0 {
			progress = clamp01(float64(completed) / float64(size))
		}

		status := StatusDownloading
		if info == nil {
			status = StatusMetadata
		} else if completed >= size && size > 0 {
			status = StatusSeeding
		}
		// A paused torrent stays in the client but shouldn't show as active.
		if c.isPaused(infoHash) {
			status = StatusPaused
		}

		// ETA: seconds remaining, based on the smoothed speed.
		var eta int64
		if status == StatusDownloading && dlSpeed > 0 && size > completed {
			eta = (size - completed) / dlSpeed
		}

		out = append(out, TorrentStatus{
			ID:            infoHash,
			InfoHash:      infoHash,
			Name:          name,
			Status:        status,
			Progress:      progress,
			DownloadSpeed: dlSpeed,
			UploadSpeed:   ulSpeed,
			Eta:           eta,
			Size:          size,
			Downloaded:    downloaded,
		})
	}
	return out
}

// Files returns the file list for a torrent (requires metadata).
func (c *Client) Files(infoHash string) ([]FileInfo, error) {
	hash, err := parseHash(infoHash)
	if err != nil {
		return nil, err
	}
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return nil, fmt.Errorf("torrent with infohash %s not found in client", infoHash)
	}
	var files []FileInfo
	for _, f := range t.Files() {
		files = append(files, FileInfo{
			Path: f.DisplayPath(),
			Size: f.Length(),
		})
	}
	return files, nil
}

// torrentName returns the display name of an active torrent ("" if not active).
func (c *Client) torrentName(infoHash string) string {
	hash, err := parseHash(infoHash)
	if err != nil {
		return ""
	}
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return ""
	}
	return t.Name()
}

// torrentSize returns the total size of an active torrent (0 if metadata absent).
func (c *Client) torrentSize(infoHash string) int64 {
	hash, err := parseHash(infoHash)
	if err != nil {
		return 0
	}
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return 0
	}
	info := t.Info()
	if info == nil {
		return 0
	}
	return info.TotalLength()
}

// MagnetFor returns the magnet link of an active torrent.
func (c *Client) MagnetFor(infoHash string) (string, error) {
	hash, err := parseHash(infoHash)
	if err != nil {
		return "", err
	}
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return "", fmt.Errorf("torrent with infohash %s not found", infoHash)
	}
	mi := t.Metainfo()
	m, err := mi.MagnetV2()
	if err != nil {
		return "", err
	}
	return m.String(), nil
}

// Close gracefully shuts down the torrent client.
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Println("[torrent] FORCE shutdown (timeout)")
			os.Exit(1)
		}
	}()

	log.Println("[torrent] Initiating graceful shutdown...")
	c.tClient.Close()
	log.Println("[torrent] Shutdown completed")
	return nil
}

// isWithinBase ensures target resolves inside baseDir (path-traversal guard for deletion).
func isWithinBase(baseDir, target string) bool {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absTarget, absBase+string(filepath.Separator))
}

// MagnetFromFile derives a magnet URI from a .torrent file on disk.
func MagnetFromFile(path string) (string, error) {
	mi, err := metainfo.LoadFromFile(path)
	if err != nil {
		return "", err
	}
	m, err := mi.MagnetV2()
	if err != nil {
		return "", err
	}
	return m.String(), nil
}
