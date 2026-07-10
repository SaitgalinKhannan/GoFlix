package torrent

import "time"

// Status represents the current state of a torrent.
type Status string

const (
	StatusMetadata    Status = "metadata" // fetching metadata
	StatusChecking    Status = "checking" // verifying local pieces against their hashes
	StatusQueued      Status = "queued"   // waiting for a download slot
	StatusDownloading Status = "downloading"
	StatusSeeding     Status = "seeding"
	StatusCompleted   Status = "completed"
	StatusPaused      Status = "paused" // stopped by user, files kept
	StatusError       Status = "error"
)

// TorrentStatus is the API representation of a torrent, matching the PLAN.md contract.
type TorrentStatus struct {
	ID            string     `json:"id"` // aliased to infoHash
	InfoHash      string     `json:"infoHash"`
	Name          string     `json:"name"`
	Status        Status     `json:"status"`
	Progress      float64    `json:"progress"`      // 0..1
	DownloadSpeed int64      `json:"downloadSpeed"` // bytes/s
	UploadSpeed   int64      `json:"uploadSpeed"`   // bytes/s
	Eta           int64      `json:"eta,omitempty"` // seconds remaining (0 = done/unknown)
	Size          int64      `json:"size"`          // bytes
	Downloaded    int64      `json:"downloaded"`    // bytes
	AddedAt       time.Time  `json:"addedAt"`       // ISO 8601
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	Error         string     `json:"error,omitempty"`
	Files         []FileInfo `json:"files,omitempty"`
}

// FileInfo describes a single file within a torrent.
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// storedTorrent is the persisted representation kept in the state manager.
// It mirrors TorrentStatus but also carries the magnet link and seeding flag
// needed to re-add torrents after a restart.
type storedTorrent struct {
	InfoHash    string     `json:"infoHash"`
	Name        string     `json:"name"`
	Magnet      string     `json:"magnet"`
	Size        int64      `json:"size"`
	AddedAt     time.Time  `json:"addedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// Seeding indicates whether the user wants this torrent active in the client.
	// false (default) means stopped/paused.
	Seeding bool `json:"seeding"`
}
