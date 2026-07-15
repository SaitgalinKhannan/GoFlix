package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from the environment.
type Config struct {
	Port               string
	TorrentsStatesFile string
	TorrentsDir        string
	PieceCompletionDir string
	SpeedLimitsFile    string
	DownloadLimit      int64 // bytes/s, 0 means unlimited
	UploadLimit        int64 // bytes/s, 0 means unlimited
	// Auth (JWT access/refresh)
	JWTSecret              string
	AuthUsername           string
	AuthPassword           string // plaintext; hashed at startup
	JellyfinURL            string
	JellyfinToken          string
	MaxConcurrentDownloads int
}

// LoadConfig loads configuration from a .env file (if present) and environment variables.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to environment variables or defaults.")
	}

	cfg := &Config{
		Port:                   os.Getenv("PORT"),
		TorrentsStatesFile:     os.Getenv("TORRENTS_STATES_FILE"),
		TorrentsDir:            os.Getenv("TORRENTS_DIR"),
		PieceCompletionDir:     os.Getenv("PIECE_COMPLETION_DIR"),
		SpeedLimitsFile:        os.Getenv("SPEED_LIMITS_FILE"),
		DownloadLimit:          envKiBRate("DOWNLOAD_RATE_LIMIT_KIB"),
		UploadLimit:            envKiBRate("UPLOAD_RATE_LIMIT_KIB"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		AuthUsername:           os.Getenv("AUTH_USERNAME"),
		AuthPassword:           os.Getenv("AUTH_PASSWORD"),
		JellyfinURL:            os.Getenv("JELLYFIN_URL"),
		JellyfinToken:          os.Getenv("JELLYFIN_TOKEN"),
		MaxConcurrentDownloads: envInt("MAX_CONCURRENT_DOWNLOADS", 3),
	}

	if cfg.Port == "" {
		cfg.Port = "8081"
	}
	if cfg.TorrentsStatesFile == "" {
		cfg.TorrentsStatesFile = "/app/data/torrent_states.json"
	}
	if cfg.TorrentsDir == "" {
		cfg.TorrentsDir = "/app/data/torrents"
	}
	if cfg.PieceCompletionDir == "" {
		cfg.PieceCompletionDir = "/app/data/torrent_data"
	}
	if cfg.SpeedLimitsFile == "" {
		cfg.SpeedLimitsFile = "/app/data/speed_limits.json"
	}
	if cfg.MaxConcurrentDownloads < 1 {
		cfg.MaxConcurrentDownloads = 1
	}

	// Auth is required for the API to function.
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required (generate one, e.g. openssl rand -hex 32)")
	}
	if cfg.AuthUsername == "" || cfg.AuthPassword == "" {
		return nil, fmt.Errorf("AUTH_USERNAME and AUTH_PASSWORD are required")
	}

	return cfg, nil
}

func envKiBRate(key string) int64 {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 || n > (1<<63-1)/1024 {
		return 0
	}
	return n * 1024
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
