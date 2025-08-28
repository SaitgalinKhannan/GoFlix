package handlers

import (
	"log"
	"net/http"
)

func HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			// Это НЕ критично для health check
			log.Printf("[health] NON-CRITICAL: Failed to write response: %v", err)
		}
	}
}
