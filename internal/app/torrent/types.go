package torrent

import "time"

// State представляет состояние торрента
type State int

const (
	StateDownloading State = iota
	StateQueued
	StateCompleted
	StatePaused
)

type ConvertingState int

const (
	StateNotConverted     ConvertingState = iota // Не конвертировано
	StateConvertingQueued                        // В очереди на конвертацию
	StateConverting                              // В процессе конвертации
	StateConverted                               // Успешно конвертирован
	StateConvertingError                         // Ошибка при конвертации
)

// Torrent представляет информация о торренте
type Torrent struct {
	InfoHash           string          `json:"infoHash"`
	Name               string          `json:"name"`
	Magnet             string          `json:"magnet"`
	Size               int64           `json:"size"`
	Done               bool            `json:"done"`
	State              State           `json:"state"`
	ConvertingState    ConvertingState `json:"convertingState"`
	CompletedAt        *time.Time      `json:"completedAt,omitempty"`
	ConvertingQueuedAt *time.Time      `json:"convertingQueuedAt,omitempty"`
	ConvertedAt        *time.Time      `json:"convertedAt,omitempty"`
	LastChecked        time.Time       `json:"lastChecked"`
	DownloadedPercent  float32         `json:"downloadedPercent"`
}

// WasCompleted проверяет, был ли торрент завершен ранее
func (t *Torrent) WasCompleted() bool {
	return t.State >= StateCompleted
}
