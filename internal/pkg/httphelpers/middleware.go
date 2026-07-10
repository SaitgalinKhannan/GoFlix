package httphelpers

import (
	"log"
	"net/http"
)

// ErrorHandler wraps the ResponseWriter to capture write errors and recover
// from panics. It forwards Flush/Unwrap so streaming responses (SSE) and
// http.ResponseController keep working through the wrapper.
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}

		defer func() {
			if err := recover(); err != nil {
				log.Printf("[http] Panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(rw, r)

		if rw.writeError != nil {
			log.Printf("[http] Write error: %v", rw.writeError)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	writeError error
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.writeError = err
	return n, err
}

// Flush forwards to the underlying writer when it supports flushing (e.g. SSE).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the underlying writer so http.ResponseController and type
// assertions can drill through this wrapper.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
