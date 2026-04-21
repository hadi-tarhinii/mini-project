package middleware

import (
	"time"
	"log"
	"net/http"

)
type responseWriter struct {
	http.ResponseWriter
	status      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the original writer
		wrappedWriter := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		log.Printf(
			"[%d] %s %s | Latency: %v",
			wrappedWriter.status,
			r.Method,
			r.URL.Path,
			time.Since(start),
		)
	})
}