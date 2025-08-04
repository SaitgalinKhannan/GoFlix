package httphelpers

import (
	"log"
	"net/http"
)

func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Оборачиваем writer для отслеживания ошибок записи
		rw := &responseWriter{ResponseWriter: w}

		defer func() {
			if err := recover(); err != nil {
				log.Printf("[http] Panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(rw, r)

		// Логируем ошибки записи
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
