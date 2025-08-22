package torrent

import (
	"GoFlix/internal/app/media"
	"GoFlix/internal/pkg/filehelpers"
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

type Client struct {
	tClient *torrent.Client
	baseDir string
}

func NewClient(clientBaseDir string, pieceCompletionDir string) (*Client, error) {
	config := torrent.NewDefaultClientConfig()

	if err := os.MkdirAll(clientBaseDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(pieceCompletionDir, 0o700); err != nil {
		return nil, err
	}

	pieceCompletion, err := storage.NewDefaultPieceCompletionForDir(pieceCompletionDir)
	if err != nil {
		return nil, err
	}

	opts := storage.NewFileClientOpts{
		ClientBaseDir:   clientBaseDir,
		PieceCompletion: pieceCompletion,
	}
	storageClient := storage.NewFileOpts(opts)
	config.DefaultStorage = storageClient

	tClient, err := torrent.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		tClient: tClient,
		baseDir: clientBaseDir,
	}, nil
}

// getClientBaseDir returns the base directory of the client.
func (c *Client) getClientBaseDir() string {
	return c.baseDir
}

// Add adds a torrent via magnet link or file path.
func (c *Client) Add(magnet string) (string, error) {
	var t *torrent.Torrent
	var err error

	if strings.HasPrefix(magnet, "magnet:") {
		t, err = c.tClient.AddMagnet(magnet)
	} else {
		mi, err := metainfo.LoadFromFile(magnet)
		if err != nil {
			return "", err
		}
		t, err = c.tClient.AddTorrent(mi)
	}

	if err != nil {
		return "", err
	}
	if t == nil {
		return "", fmt.Errorf("failed to add torrent: %s", magnet)
	}

	<-t.GotInfo()
	t.DownloadAll()

	return t.InfoHash().String(), nil
}

// toTorrent converts a torrent.Torrent to our local Torrent type.
func (c *Client) toTorrent(t *torrent.Torrent) (*Torrent, error) {
	if t == nil {
		return nil, errors.New("cannot convert nil torrent")
	}

	metaInfo := t.Metainfo()
	magnet, err := metaInfo.MagnetV2()
	if err != nil {
		return nil, fmt.Errorf("failed to get magnet link: %w", err)
	}

	infoHash := t.InfoHash().String()
	done := t.BytesCompleted() == t.Length()
	percent := getPercent(t.BytesCompleted(), t.Length())
	state := StateDownloading
	if done {
		state = StateCompleted
	}

	return &Torrent{
		InfoHash:          infoHash,
		Name:              t.Name(),
		Magnet:            magnet.String(),
		Size:              t.Length(),
		Done:              done,
		DownloadedPercent: percent,
		State:             state,
		ConvertingState:   StateNotConverted,
		LastChecked:       time.Now(),
	}, nil
}

// GetTorrent returns a torrent by its infoHash.
func (c *Client) GetTorrent(infoHash string) (*Torrent, error) {
	hash := metainfo.NewHashFromHex(infoHash)
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return nil, fmt.Errorf("torrent with infohash %s not found in client", infoHash)
	}
	return c.toTorrent(t)
}

// GetTorrents returns all active torrents from the client.
func (c *Client) GetTorrents() []Torrent {
	activeTorrents := c.tClient.Torrents()
	torrents := make([]Torrent, 0, len(activeTorrents))

	for _, t := range activeTorrents {
		converted, err := c.toTorrent(t)
		if err != nil {
			log.Printf("[client] error converting torrent: %v", err)
			continue
		}
		torrents = append(torrents, *converted)
	}
	return torrents
}

// GetTorrentVideoFiles returns a list of video files for a torrent.
func (c *Client) GetTorrentVideoFiles(t *torrent.Torrent) ([]string, error) {
	var videoFiles []string
	baseDir := c.getClientBaseDir()
	torrentName := t.Name()

	for _, file := range t.Files() {
		if filehelpers.IsVideoFile(file.DisplayPath()) {
			var fullPath string
			if filehelpers.IsVideoFile(torrentName) {
				fullPath = filepath.Join(baseDir, file.DisplayPath())
			} else {
				// if file in folder
				fullPath = filepath.Join(baseDir, torrentName, file.DisplayPath())
			}
			videoFiles = append(videoFiles, fullPath)
		}
	}

	return videoFiles, nil
}

// GetTorrentVideoFilesInfo retrieves information about all video files in a torrent concurrently.
func (c *Client) GetTorrentVideoFilesInfo(t *torrent.Torrent) ([]VideoFile, error) {
	if t == nil {
		return nil, errors.New("torrent is nil")
	}

	torrentVideoFiles, err := c.GetTorrentVideoFiles(t)
	if err != nil {
		return nil, err
	}
	if len(torrentVideoFiles) == 0 {
		return []VideoFile{}, nil
	}

	var wg sync.WaitGroup
	results := make([]VideoFile, len(torrentVideoFiles))

	for i, file := range torrentVideoFiles {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()

			info, err := media.GetVideoInfo(path)
			results[index] = VideoFile{
				Path:      path,
				VideoInfo: info,
				Error:     err,
			}
			if err != nil {
				log.Printf("[client] Error getting video info for %s: %v", path, err)
			}
		}(i, file)
	}
	wg.Wait()

	return results, nil
}

// PauseTorrent pauses a torrent's download.
func (c *Client) PauseTorrent(infoHash string) error {
	hash := metainfo.NewHashFromHex(infoHash)
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		return fmt.Errorf("torrent with infohash %s not found", infoHash)
	}
	t.Drop()
	return nil
}

// DeleteTorrent removes a torrent from the client.
func (c *Client) DeleteTorrent(infoHash string) error {
	hash := metainfo.NewHashFromHex(infoHash)
	t, ok := c.tClient.Torrent(hash)
	if !ok {
		// If torrent is not in client, it's not an error in this context.
		return nil
	}
	t.Drop()
	return nil
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
