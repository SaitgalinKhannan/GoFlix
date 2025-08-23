package torrent

import (
	"GoFlix/internal/app/media"
	"GoFlix/internal/pkg/filehelpers"
	"fmt"
	"log"
	"path/filepath"
)

// Service handles the business logic for managing torrents.
type Service struct {
	client       *Client
	stateManager *StateManager
}

// NewService creates a new torrent service.
func NewService(client *Client, stateManager *StateManager) *Service {
	return &Service{
		client:       client,
		stateManager: stateManager,
	}
}

// AddTorrent adds a new torrent from a magnet link or file path.
func (s *Service) AddTorrent(magnet string) (string, error) {
	return s.client.Add(magnet)
}

// GetTorrents returns a list of all torrents (active and inactive).
func (s *Service) GetTorrents() []Torrent {
	activeTorrents := s.client.GetTorrents()

	// Use a map to merge active torrents with stored states
	torrentsMap := s.stateManager.GetAllTorrents()

	for _, t := range activeTorrents {
		// Update existing torrent from state or add if not present
		if existing, ok := torrentsMap[t.InfoHash]; ok {
			// Update fields from active torrent
			existing.DownloadedPercent = t.DownloadedPercent
			existing.Done = t.Done
			if !existing.Done {
				existing.State = StateDownloading
			}
			torrentsMap[t.InfoHash] = existing
		} else {
			torrentsMap[t.InfoHash] = &t
		}
	}

	// Convert map back to slice
	torrents := make([]Torrent, 0, len(torrentsMap))

	for _, t := range torrentsMap {
		if t.Done && t.VideoFiles == nil {
			info, err := s.client.GetTorrentVideoFilesInfo(t)
			if err != nil {
				fmt.Println(err)
			}
			if info != nil {
				t.VideoFiles = info
			}
		}
		torrents = append(torrents, *t)
	}

	return torrents
}

// GetTorrent returns a single torrent by its info hash.
func (s *Service) GetTorrent(infoHash string) (*Torrent, error) {
	// First, check the state manager
	t, err := s.stateManager.GetTorrent(infoHash)
	if err == nil {
		// If it's also active in the client, update its status
		if activeTorrent, err := s.client.GetTorrent(infoHash); err == nil && activeTorrent != nil {
			t.DownloadedPercent = activeTorrent.DownloadedPercent
			t.Done = activeTorrent.Done
		}

		// обновление информации о видео файлах
		s.updateTorrentVideoFiles(t)

		return t, nil
	}

	// If not in state, check the client directly
	torrent, err := s.client.GetTorrent(infoHash)
	if err != nil || torrent == nil {
		return nil, err
	}

	// обновление информации о видео файлах
	s.updateTorrentVideoFiles(t)

	return torrent, nil
}

// PauseTorrent pauses a torrent.
func (s *Service) PauseTorrent(infoHash string) error {
	if err := s.client.PauseTorrent(infoHash); err != nil {
		return err
	}
	return s.stateManager.MarkAsPaused(infoHash)
}

// ResumeTorrent resumes a torrent.
func (s *Service) ResumeTorrent(infoHash string) error {
	torrent, err := s.stateManager.GetTorrent(infoHash)
	if err != nil {
		return fmt.Errorf("torrent with infohash %s not found", infoHash)
	}

	if _, err := s.client.Add(torrent.Magnet); err != nil {
		return fmt.Errorf("[service] failed to resume torrent: %v", err)
	}
	return s.stateManager.MarkAsResumed(infoHash)
}

// DeleteTorrent deletes a torrent.
func (s *Service) DeleteTorrent(infoHash string) error {
	if err := s.client.DeleteTorrent(infoHash); err != nil {
		// Log error but continue to remove from state
		log.Printf("[service] error dropping torrent from client: %v", err)
	}

	s.stateManager.RemoveTorrent(infoHash)
	return nil
}

// ConvertTorrent adds a torrent to the conversion queue.
func (s *Service) ConvertTorrent(infoHash string) error {
	torrent, err := s.stateManager.GetTorrent(infoHash)
	if err != nil {
		return fmt.Errorf("torrent with infohash %s not found", infoHash)
	}

	if err := s.stateManager.MarkAsQueued(infoHash); err != nil {
		return err
	}

	// Add to the conversion queue
	select {
	case s.stateManager.conversionQueue <- torrent:
		log.Printf("Added torrent to conversion queue: %s", torrent.Name)
	default:
		log.Printf("Conversion queue is full, torrent: %s", torrent.Name)
		return fmt.Errorf("conversion queue is full")
	}

	return nil
}

func (s *Service) ConvertTorrentToHls(t *Torrent) error {
	if len(t.VideoFiles) == 0 {
		s.updateTorrentVideoFiles(t)
	}

	if t.VideoFiles != nil {
		for _, f := range t.VideoFiles {
			if !filehelpers.IsVideoFile(f.Path) {
				continue
			}

			abs, err := filepath.Abs(f.Path)
			if err != nil {
				return err
			}
			err = media.ConvertToHls(abs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// updateTorrentVideoFiles обновляет информацию о видеофайлах торрента, если она отсутствует и торрент завершён.
func (s *Service) updateTorrentVideoFiles(t *Torrent) {
	if t.Done && t.VideoFiles == nil {
		info, err := s.client.GetTorrentVideoFilesInfo(t)
		if err != nil {
			fmt.Println("Error fetching video files info:", err)
			return
		}
		if info != nil {
			t.VideoFiles = info
		}
	}
}
