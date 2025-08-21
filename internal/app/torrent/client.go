package torrent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

var _ io.Closer = (*Client)(nil)

type Client struct {
	tClient      *torrent.Client
	StateManager *StateManager
}

func NewClient(clientBaseDir string, pieceCompletionDir string, stateFile string) (*Client, error) {
	config := torrent.NewDefaultClientConfig()

	err := os.MkdirAll(clientBaseDir, 0o700)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(pieceCompletionDir, 0o700)
	if err != nil {
		return nil, err
	}

	pieceCompletion, err := storage.NewDefaultPieceCompletionForDir(pieceCompletionDir)
	if err != nil {
		return nil, err
	}

	opts := storage.NewFileClientOpts{
		ClientBaseDir:   clientBaseDir, // файлы торрентов будут храниться здесь
		PieceCompletion: pieceCompletion,
	}
	// Создаем клиент хранения с настроенными путями
	storageClient := storage.NewFileOpts(opts)
	config.DefaultStorage = storageClient

	tClient, err := torrent.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		tClient:      tClient,
		StateManager: NewTorrentStateManager(stateFile),
	}, nil
}

// Add Добавляем торрент через magnet-ссылку
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

	// Ждем метаданные (критично для magnet-ссылок)
	<-t.GotInfo()
	t.DownloadAll()

	infoHash := t.InfoHash().String()

	return infoHash, nil
}

// GetTorrentInfo convert *torrent.Torrent to Torrent
func (c *Client) GetTorrentInfo(t *torrent.Torrent) (*Torrent, error) {
	if t != nil {
		metaInfo := t.Metainfo()
		magnet, err := metaInfo.MagnetV2()

		if err != nil {
			return nil, fmt.Errorf("[client] torrent's magnet not found:%v", err)
		}

		infoHash := t.InfoHash().String()
		percent := getPercent(t.BytesCompleted(), t.Length())
		newTorrent := Torrent{
			InfoHash:          infoHash,
			Name:              t.Name(),
			Magnet:            magnet.String(),
			Size:              t.Length(),
			Done:              int(percent) == 100,
			DownloadedPercent: percent,
			State:             StateDownloading,
			ConvertingState:   StateNotConverted,
			LastChecked:       time.Now(),
		}

		// get torrent state
		oldTorrent, exist := c.StateManager.states[infoHash]
		if exist && oldTorrent != nil && newTorrent.Done {
			newTorrent.State = oldTorrent.State
			newTorrent.ConvertingState = oldTorrent.ConvertingState
			newTorrent.CompletedAt = oldTorrent.CompletedAt
			newTorrent.ConvertingQueuedAt = oldTorrent.ConvertingQueuedAt
			newTorrent.ConvertedAt = oldTorrent.ConvertedAt
		}

		return &newTorrent, nil
	}

	return nil, errors.New("[client] torrent not found")
}

// GetTorrent return torrent by infoHash
func (c *Client) GetTorrent(infoHash string) (*Torrent, error) {
	torrents := c.tClient.Torrents()

	for _, t := range torrents {
		if t.InfoHash().String() == infoHash {
			convertedTorrent, err := c.GetTorrentInfo(t)

			if err != nil {
				return nil, err
			}

			if convertedTorrent != nil {
				return convertedTorrent, nil
			}
		}
	}

	return nil, nil
}

/*func (c *Client) GetTorrents() ([]Torrent, error) {
	ts := c.tClient.Torrents()
	torrents := make([]Torrent, 0, len(ts))

	for _, t := range ts {
		newTorrent, err := c.GetTorrentInfo(t)

		if err != nil {
			log.Println(err)
		}

		if newTorrent != nil {
			torrents = append(torrents, *newTorrent)
		}
	}

	return torrents, nil
}*/

func (c *Client) GetTorrents() []Torrent {
	ts := c.tClient.Torrents()
	// tsFromState is a map where the key is the InfoHash. This is very useful.
	tsFromState := c.StateManager.GetAllTorrents()

	// Pre-allocate slice capacity for better performance, assuming most torrents will be active.
	torrents := make([]Torrent, 0, len(ts)+len(tsFromState))

	// First, add all the currently active torrents from the client.
	for _, t := range ts {
		newTorrent, err := c.GetTorrentInfo(t)
		if err != nil {
			log.Printf("[client] error getting torrent info: %v", err)
			continue // Skip to the next torrent
		}
		if newTorrent != nil {
			torrents = append(torrents, *newTorrent)
			// Since we've found this torrent, remove it from the state map.
			// This way, tsFromState will only contain torrents that are not currently active.
			delete(tsFromState, newTorrent.InfoHash)
		}
	}

	// Now, add the remaining torrents from the state that were not in the active list.
	// The loop will iterate over what's left in the map.
	for _, stateTorrent := range tsFromState {
		torrents = append(torrents, *stateTorrent)
	}

	return torrents
}

func (c *Client) PauseTorrent(infoHash string) error {
	torrents := c.tClient.Torrents()

	for _, t := range torrents {
		if t.InfoHash().String() == infoHash {
			t.Drop()
			return c.StateManager.MarkAsPaused(infoHash)
		}
	}

	return nil
}

func (c *Client) ResumeTorrent(infoHash string) error {
	torrents := c.GetTorrents()

	for _, t := range torrents {
		if t.InfoHash == infoHash {
			if _, err := c.Add(t.Magnet); err != nil {
				return fmt.Errorf("[client] Failed to add torrent to client: %v\n", err)
			}
			return c.StateManager.MarkAsResumed(infoHash)
		}
	}

	return nil
}

func (c *Client) DeleteTorrent(infoHash string) error {
	torrents := c.tClient.Torrents()

	for _, t := range torrents {
		if t.InfoHash().String() == infoHash {
			t.Drop()
		}
	}

	return nil
}

func (c *Client) ConvertTorrent(infoHash string) error {
	torrents := c.GetTorrents()

	for _, t := range torrents {
		if t.InfoHash == infoHash {
			if err := c.StateManager.MarkAsQueued(t.InfoHash); err != nil {
				return fmt.Errorf("error marking torrent as queued: %v", err)
			} else {
				// Добавляем в очередь конвертации
				select {
				case c.StateManager.conversionQueue <- &t:
					log.Printf("Added torrent to conversion queue: %s", t.Name)
				default:
					log.Printf("Conversion queue is full, torrent: %s", t.Name)
					return fmt.Errorf("torrent conversion queue is full")
				}
			}
			break
		}
	}

	return nil
}

// Close Закрывает торрент-клиент
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Println("[torrent] FORCE shutdown (timeout)")
			os.Exit(1) // Крайний случай
		}
	}()

	log.Println("[torrent] Initiating graceful shutdown...")

	// 1. Останавливаем все торренты
	for _, t := range c.tClient.Torrents() {
		log.Printf("[torrent] Removing %s", t.InfoHash().String())
		t.Drop() // Корректная остановка торрента
	}

	// 2. Закрываем клиент
	if err := c.tClient.Close(); err != nil {
		log.Printf("[torrent] Error during shutdown: %v", err)
	}

	log.Println("[torrent] Shutdown completed")

	return nil
}
