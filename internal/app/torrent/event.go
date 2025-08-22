package torrent

import (
	"log"
	"time"
)

// Event TorrentEvent событие торрента
type Event struct {
	Type      string    `json:"type"`
	Torrent   *Torrent  `json:"torrent"`
	Timestamp time.Time `json:"timestamp"`
}

// EventHandler обработчик событий
type EventHandler struct {
	service *Service
}

func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

// Start запускает обработку событий
func (eh *EventHandler) Start() {
	go func() {
		for event := range eh.service.stateManager.EventChannel() {
			eh.handleEvent(event)
		}
	}()
}

func (eh *EventHandler) handleEvent(event Event) {
	switch event.Type {
	case "torrent_loaded":
		log.Printf("Processing torrent: %s", event.Torrent.Name)

		// Добавляем торрент в клиент
		if _, err := eh.service.AddTorrent(event.Torrent.Magnet); err != nil {
			log.Printf("Failed to add torrent to client: %v\n", err)
		} else {
			log.Printf("Successfully added torrent to client: %s\n", event.Torrent.Name)
		}

	case "download_completed":
		log.Printf("Torrent download completed: %s", event.Torrent.Name)

		// Проверяем, не был ли уже обработан
		if !eh.service.stateManager.IsAlreadyProcessed(event.Torrent.InfoHash) {
			// Добавляем в очередь на конвертацию
			if err := eh.service.ConvertTorrent(event.Torrent.InfoHash); err != nil {
				log.Printf("Error queuing torrent for conversion: %v", err)
			}
		} else {
			log.Printf("Torrent already processed: %s", event.Torrent.Name)
		}

	case "downloading_paused":
		log.Printf("Torrent downloading paused: %s", event.Torrent.Name)

	case "downloading_resumed":
		log.Printf("Torrent downloading resumed: %s", event.Torrent.Name)

	case "queued_for_conversion":
		log.Printf("Torrent queued for conversion: %s", event.Torrent.Name)

	case "conversion_completed":
		log.Printf("Torrent conversion completed: %s", event.Torrent.Name)
		// Video file info is updated on demand, so we don't need to do anything here.

	default:
		log.Printf("Unknown event type: %s", event.Type)
	}
}

// GetConversionQueue возвращает канал очереди конвертации
func (eh *EventHandler) GetConversionQueue() <-chan *Torrent {
	return eh.service.stateManager.conversionQueue
}

// Stop останавливает обработчик событий
func (eh *EventHandler) Stop() {
	close(eh.service.stateManager.conversionQueue)
}
