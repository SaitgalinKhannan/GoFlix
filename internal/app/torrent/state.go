package torrent

import (
	"GoFlix/internal/pkg/filehelpers"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// StateManager управляет состояниями торрентов
type StateManager struct {
	mu           sync.RWMutex
	states       map[string]*Torrent
	stateFile    string
	eventChannel chan Event

	// Каналы для фоновых операций
	saveChannel  chan struct{}
	batchUpdates chan *Torrent

	// Контроль фоновых процессов
	saveInProgress sync.Mutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewTorrentStateManager создает новый менеджер состояний
func NewTorrentStateManager(stateFile string) *StateManager {
	sm := &StateManager{
		states:       make(map[string]*Torrent),
		stateFile:    stateFile,
		eventChannel: make(chan Event, 100),
		saveChannel:  make(chan struct{}, 1),
		batchUpdates: make(chan *Torrent, 100),
		stopChan:     make(chan struct{}),
	}

	// Загружаем существующие состояния
	if err := sm.loadStates(); err != nil {
		log.Printf("Warning: failed to load states: %v", err)
	}

	// Запускаем фоновые процессы
	sm.startBackgroundProcesses()

	return sm
}

// loadStates загружает состояния из файла
func (sm *StateManager) loadStates() error {
	file, err := os.Open(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("State file does not exist, starting with empty state")
			return nil
		}
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer filehelpers.CloseFile(file)

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&sm.states); err != nil {
		return fmt.Errorf("failed to decode states: %w", err)
	}

	log.Printf("Loaded %d torrent states from file", len(sm.states))

	// Отправляем события для всех загруженных торрентов
	sm.sendLoadedTorrentEvents()

	return nil
}

// sendLoadedTorrentEvents отправляет события для всех загруженных торрентов
func (sm *StateManager) sendLoadedTorrentEvents() {
	for _, torrent := range sm.states {
		event := Event{
			Type:      "torrent_loaded",
			Torrent:   torrent,
			Timestamp: time.Now(),
		}

		select {
		case sm.eventChannel <- event:
			log.Printf("Sent loaded event for torrent: %s", torrent.Name)
		default:
			log.Printf("Event channel is full, dropping loaded event for: %s", torrent.Name)
		}
	}
}

// saveStates сохраняет состояния в файл
func (sm *StateManager) saveStates() error {
	sm.saveInProgress.Lock()
	defer sm.saveInProgress.Unlock()

	sm.mu.RLock()
	statesCopy := make(map[string]*Torrent, len(sm.states))
	for k, v := range sm.states {
		statesCopy[k] = v
	}
	sm.mu.RUnlock()

	file, err := os.Create(sm.stateFile + ".tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(statesCopy); err != nil {
		filehelpers.CloseFile(file)
		filehelpers.OsRemove(sm.stateFile + ".tmp")
		return fmt.Errorf("failed to encode states: %w", err)
	}

	if err := file.Close(); err != nil {
		filehelpers.OsRemove(sm.stateFile + ".tmp")
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(sm.stateFile+".tmp", sm.stateFile); err != nil {
		filehelpers.OsRemove(sm.stateFile + ".tmp")
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// startBackgroundProcesses запускает фоновые процессы
func (sm *StateManager) startBackgroundProcesses() {
	sm.wg.Add(2)

	// Процесс периодического сохранения
	go func() {
		defer sm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sm.stopChan:
				return
			case <-ticker.C:
				if err := sm.saveStates(); err != nil {
					log.Printf("Error saving states: %v", err)
				}
			case <-sm.saveChannel:
				if err := sm.saveStates(); err != nil {
					log.Printf("Error saving states: %v", err)
				}
			}
		}
	}()

	// Процесс обработки пакетных обновлений
	go func() {
		defer sm.wg.Done()
		for {
			select {
			case <-sm.stopChan:
				return
			case torrent := <-sm.batchUpdates:
				sm.updateTorrentState(torrent)
			}
		}
	}()
}

// updateTorrentState обновляет состояние торрента
func (sm *StateManager) updateTorrentState(torrent *Torrent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldTorrent, exists := sm.states[torrent.InfoHash]

	// if torrent not downloaded drop all states
	if !torrent.Done {
		torrent.State = StateDownloading
		torrent.CompletedAt = nil
		torrent.ConvertingQueuedAt = nil
		torrent.ConvertedAt = nil
	}

	sm.states[torrent.InfoHash] = torrent

	// Проверяем, если торрент только что завершился или его нет в списке и он завершился
	if (oldTorrent != nil && !oldTorrent.Done && torrent.Done) || (!exists && torrent.Done) {
		torrent.State = StateCompleted
		now := time.Now()
		torrent.CompletedAt = &now

		// Отправляем событие о завершении загрузки
		event := Event{
			Type:      "download_completed",
			Torrent:   torrent,
			Timestamp: now,
		}

		select {
		case sm.eventChannel <- event:
		default:
			log.Println("Event channel is full, dropping event")
		}
	}

	// Запланировать сохранение
	select {
	case sm.saveChannel <- struct{}{}:
	default:
	}
}

// GetTorrent получает торрент по хешу
func (sm *StateManager) GetTorrent(infoHash string) (*Torrent, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	torrent, exists := sm.states[infoHash]
	return torrent, exists
}

// GetAllTorrents возвращает все торренты
func (sm *StateManager) GetAllTorrents() map[string]*Torrent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Torrent, len(sm.states))
	for k, v := range sm.states {
		result[k] = v
	}
	return result
}

// UpdateTorrent обновляет информацию о торренте
func (sm *StateManager) UpdateTorrent(torrent *Torrent) {
	torrent.LastChecked = time.Now()

	select {
	case sm.batchUpdates <- torrent:
	default:
		log.Println("Batch updates channel is full")
	}
}

// MarkAsQueued помечает торрент как добавленный в очередь на конвертацию
func (sm *StateManager) MarkAsQueued(infoHash string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	torrent, exists := sm.states[infoHash]
	if !exists {
		return fmt.Errorf("torrent %s not found", infoHash)
	}

	if torrent.State != StateCompleted {
		return fmt.Errorf("torrent %s is not completed yet", infoHash)
	}

	if torrent.ConvertingState >= StateConvertingQueued && torrent.ConvertingState != StateConvertingError {
		return fmt.Errorf("torrent %s is already queued or processed", infoHash)
	}

	torrent.ConvertingState = StateConvertingQueued
	now := time.Now()
	torrent.ConvertingQueuedAt = &now
	torrent.LastChecked = now

	// Сохраняем состояние
	select {
	case sm.saveChannel <- struct{}{}:
	default:
	}

	// Отправляем событие
	event := Event{
		Type:      "queued_for_conversion",
		Torrent:   torrent,
		Timestamp: now,
	}

	select {
	case sm.eventChannel <- event:
	default:
		log.Println("Event channel is full, dropping event")
	}

	return nil
}

// MarkAsConverting помечает торрент как конвертируемый
func (sm *StateManager) MarkAsConverting(infoHash string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	torrent, exists := sm.states[infoHash]
	if !exists {
		return fmt.Errorf("torrent %s not found", infoHash)
	}

	torrent.ConvertingState = StateConverting
	torrent.LastChecked = time.Now()

	// Сохраняем состояние
	select {
	case sm.saveChannel <- struct{}{}:
	default:
	}

	return nil
}

// MarkAsConverted помечает торрент как успешно конвертированный
func (sm *StateManager) MarkAsConverted(infoHash string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	torrent, exists := sm.states[infoHash]
	if !exists {
		return fmt.Errorf("torrent %s not found", infoHash)
	}

	torrent.ConvertingState = StateConverted
	now := time.Now()
	torrent.ConvertedAt = &now
	torrent.LastChecked = now

	// Сохраняем состояние
	select {
	case sm.saveChannel <- struct{}{}:
	default:
	}

	// Отправляем событие
	event := Event{
		Type:      "conversion_completed",
		Torrent:   torrent,
		Timestamp: now,
	}

	select {
	case sm.eventChannel <- event:
	default:
		log.Println("Event channel is full, dropping event")
	}

	return nil
}

// MarkAsError помечает торрент с ошибкой конвертации
func (sm *StateManager) MarkAsError(infoHash string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	torrent, exists := sm.states[infoHash]
	if !exists {
		return fmt.Errorf("torrent %s not found", infoHash)
	}

	torrent.ConvertingState = StateConvertingError
	torrent.LastChecked = time.Now()

	// Сохраняем состояние
	select {
	case sm.saveChannel <- struct{}{}:
	default:
	}

	return nil
}

// IsAlreadyProcessed проверяет, был ли торрент уже обработан
func (sm *StateManager) IsAlreadyProcessed(infoHash string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	torrent, exists := sm.states[infoHash]
	if !exists {
		return false
	}

	return torrent.ConvertingState >= StateConvertingQueued && torrent.ConvertingState != StateConvertingError
}

// EventChannel возвращает канал событий
func (sm *StateManager) EventChannel() <-chan Event {
	return sm.eventChannel
}

// Stop останавливает фоновые процессы
func (sm *StateManager) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()

	// Финальное сохранение
	if err := sm.saveStates(); err != nil {
		log.Printf("Error during final save: %v\n", err)
	}

	close(sm.eventChannel)
}
