package filehelpers

import (
	"path/filepath"
	"strings"
)

func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	videoExtensions := []string{
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm",
		".m4v", ".3gp", ".ogv", ".ts", ".mts", ".m2ts", ".vob",
		".asf", ".rm", ".rmvb", ".divx", ".xvid", ".f4v", ".mpg",
		".mpeg", ".m1v", ".m2v", ".mpe", ".mpv", ".dv", ".qt",
	}

	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true
		}
	}

	return false
}
