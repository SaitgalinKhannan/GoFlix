package handlers

import (
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StarfieldHandler serves the starfield visualization page and static assets
func StarfieldHandler(staticDir string) http.HandlerFunc {
	// Ensure common MIME types are registered
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")

	return func(w http.ResponseWriter, r *http.Request) {
		// If requesting the root path, serve the starfield.html file
		if r.URL.Path == "/starfield/" || r.URL.Path == "/starfield" {
			http.ServeFile(w, r, filepath.Join(staticDir, "starfield.html"))
			return
		}

		// For other paths, strip the /starfield prefix and serve static files
		// Remove the /starfield prefix from the path
		filePath := strings.TrimPrefix(r.URL.Path, "/starfield")

		// If the path is empty after stripping, serve the main page
		if filePath == "" || filePath == "/" {
			http.ServeFile(w, r, filepath.Join(staticDir, "starfield.html"))
			return
		}

		// Construct the full file path
		fullPath := filepath.Join(staticDir, filePath)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Return 404 for missing files instead of serving the main page
			http.NotFound(w, r)
			return
		}

		// Set content type based on file extension
		ext := filepath.Ext(fullPath)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		// Serve the file
		file, err := os.Open(fullPath)
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			log.Printf("Error opening file %s: %v", fullPath, err)
			return
		}
		defer file.Close()

		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Error serving file %s: %v", fullPath, err)
		}
	}
}
