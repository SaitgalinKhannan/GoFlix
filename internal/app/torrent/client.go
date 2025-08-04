package torrent

import (
	"context"
	"errors"
	"fmt"
	"github.com/anacrolix/torrent/storage"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
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

// Add Добавляем торрент через magnet-ссылку или .torrent файл
func (c *Client) Add(magnetOrFilePath string) (string, error) {
	var t *torrent.Torrent
	var err error

	if strings.HasPrefix(magnetOrFilePath, "magnet:") {
		t, err = c.tClient.AddMagnet(magnetOrFilePath)
	} else {
		mi, err := metainfo.LoadFromFile(magnetOrFilePath)
		if err != nil {
			return "", err
		}
		t, err = c.tClient.AddTorrent(mi)
	}

	if err != nil {
		return "", err
	}

	if t == nil {
		return "", fmt.Errorf("failed to add torrent: %s", magnetOrFilePath)
	}

	// Ждем метаданные (критично для magnet-ссылок)
	<-t.GotInfo()
	t.DownloadAll()

	infoHash := t.InfoHash().String()

	return infoHash, nil
}

// GetTorrent convert *torrent.Torrent to Torrent
func (c *Client) GetTorrent(t *torrent.Torrent) (*Torrent, error) {
	if t != nil {
		metaInfo := t.Metainfo()
		magnet, err := metaInfo.MagnetV2()

		if err != nil {
			return nil, fmt.Errorf("[client] torrent's magnet not found:%v\n", err)
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

func (c *Client) GetTorrents() ([]Torrent, error) {
	ts := c.tClient.Torrents()
	var torrents []Torrent

	if ts != nil {
		for _, t := range ts {
			newTorrent, err := c.GetTorrent(t)

			if err != nil {
				log.Println(err)
			}

			if newTorrent != nil {
				torrents = append(torrents, *newTorrent)
			}
		}
	}

	return torrents, nil
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
