package middleware

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Loggable struct {
	logger *zap.Logger
}

func NewLoggable(logger *zap.Logger) *Loggable {
	return &Loggable{
		logger: logger,
	}
}

type enrichedResponseWriter struct {
	http.ResponseWriter

	Status      int
	ContentSize int
}

func newEnrichedResponseWriter(w http.ResponseWriter) *enrichedResponseWriter {
	return &enrichedResponseWriter{
		ResponseWriter: w,
	}
}

func (w *enrichedResponseWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *enrichedResponseWriter) Write(b []byte) (int, error) {
	w.ContentSize = len(b)
	return w.ResponseWriter.Write(b)
}

func (l *Loggable) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(logger *zap.Logger) {
			_ = logger.Sync()
		}(l.logger)

		sl := l.logger.Sugar()
		sl.Infow("Request received",
			"url", r.URL.String(),
			"method", r.Method)

		ew := newEnrichedResponseWriter(w)
		start := time.Now()
		next.ServeHTTP(ew, r)
		duration := time.Since(start)

		sl.Infow("Request processed",
			"url", r.URL.String(),
			"method", r.Method,
			"status", ew.Status,
			"responseSize", ew.ContentSize,
			"durationMs", duration.Milliseconds())
	})
}
