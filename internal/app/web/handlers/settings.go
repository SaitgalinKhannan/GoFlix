package handlers

import (
	"GoFlix/internal/app/torrent"
	"encoding/json"
	"net/http"
)

// GetSpeedLimitsHandler returns global transfer limits in bytes per second.
func GetSpeedLimitsHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, service.SpeedLimits())
	}
}

// SetSpeedLimitsHandler updates global transfer limits without restarting the client.
func SetSpeedLimitsHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
		decoder.DisallowUnknownFields()
		var limits torrent.SpeedLimits
		if err := decoder.Decode(&limits); err != nil {
			http.Error(w, "invalid speed limits: "+err.Error(), http.StatusBadRequest)
			return
		}
		if limits.Download < 0 || limits.Upload < 0 {
			http.Error(w, "speed limits cannot be negative", http.StatusBadRequest)
			return
		}
		if err := service.SetSpeedLimits(limits); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, service.SpeedLimits())
	}
}
