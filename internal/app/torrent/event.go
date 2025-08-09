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
	stateManager *StateManager
}

func NewEventHandler(sm *StateManager) *EventHandler {
	return &EventHandler{
		stateManager: sm,
	}
}

// Start запускает обработку событий
func (eh *EventHandler) Start(client *Client) {
	go func() {
		for event := range eh.stateManager.EventChannel() {
			eh.handleEvent(event, client)
		}
	}()
}

func (eh *EventHandler) handleEvent(event Event, client *Client) {
	switch event.Type {
	case "torrent_loaded":
		log.Printf("Processing torrent: %s", event.Torrent.Name)

		// Добавляем торрент в клиент
		if _, err := client.Add(event.Torrent.Magnet); err != nil {
			log.Printf("Failed to add torrent to client: %v\n", err)
		} else {
			log.Printf("Successfully added torrent to client: %s\n", event.Torrent.Name)
		}

	case "download_completed":
		log.Printf("Torrent download completed: %s", event.Torrent.Name)

		// Проверяем, не был ли уже обработан
		if !eh.stateManager.IsAlreadyProcessed(event.Torrent.InfoHash) {
			// Добавляем в очередь на конвертацию
			if err := eh.stateManager.MarkAsQueued(event.Torrent.InfoHash); err != nil {
				log.Printf("Error marking torrent as queued: %v", err)
			} else {
				// Добавляем в очередь конвертации
				select {
				case eh.stateManager.conversionQueue <- event.Torrent:
					log.Printf("Added torrent to conversion queue: %s", event.Torrent.Name)
				default:
					log.Printf("Conversion queue is full, torrent: %s", event.Torrent.Name)
				}
			}
		} else {
			log.Printf("Torrent already processed: %s", event.Torrent.Name)
		}

	case "queued_for_conversion":
		log.Printf("Torrent queued for conversion: %s", event.Torrent.Name)

	case "conversion_completed":
		log.Printf("Torrent conversion completed: %s", event.Torrent.Name)

	case "downloading_paused":
		log.Printf("Torrent downloading paused: %s", event.Torrent.Name)

	case "downloading_resumed":
		log.Printf("Torrent downloading resumed: %s", event.Torrent.Name)

	default:
		log.Printf("Unknown event type: %s", event.Type)
	}
}

// GetConversionQueue возвращает канал очереди конвертации
func (eh *EventHandler) GetConversionQueue() <-chan *Torrent {
	return eh.stateManager.conversionQueue
}

// Stop останавливает обработчик событий
func (eh *EventHandler) Stop() {
	close(eh.stateManager.conversionQueue)
}
