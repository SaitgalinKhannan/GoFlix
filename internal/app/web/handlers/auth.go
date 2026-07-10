package handlers

import (
	"GoFlix/internal/app/auth"
	"encoding/json"
	"net/http"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// LoginHandler authenticates the user and returns an access/refresh token pair.
func LoginHandler(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "username and password are required")
			return
		}
		if err := svc.VerifyCredentials(req.Username, req.Password); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		pair, err := svc.Issue()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}
		writeJSON(w, http.StatusOK, pair)
	}
}

// RefreshHandler exchanges a valid refresh token for a new access/refresh pair.
func RefreshHandler(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.RefreshToken == "" {
			writeJSONError(w, http.StatusBadRequest, "refreshToken is required")
			return
		}
		if _, err := svc.Verify(req.RefreshToken, auth.RefreshTokenKind()); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		pair, err := svc.Issue()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}
		writeJSON(w, http.StatusOK, pair)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
