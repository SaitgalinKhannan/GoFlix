package handlers

import (
	"GoFlix/internal/app/torrent"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SSEHandler streams torrent status updates via Server-Sent Events.
func SSEHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

		ch, snapshot := service.Subscribe()
		defer service.Unsubscribe(ch)

		// Send the current snapshot immediately.
		if len(snapshot) > 0 {
			writeSSE(w, snapshot)
			flusher.Flush()
		}

		// Heartbeat keeps the connection alive through proxies.
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		ctx := r.Context()
		for {
			select {
			case data, ok := <-ch:
				if !ok {
					return
				}
				if err := writeSSE(w, data); err != nil {
					log.Printf("[sse] write error: %v", err)
					return
				}
				flusher.Flush()
			case <-heartbeat.C:
				if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, data []byte) error {
	if _, err := fmt.Fprintf(w, "event: status\ndata: %s\n\n", data); err != nil {
		return err
	}
	return nil
}
