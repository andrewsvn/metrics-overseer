package middleware

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

type HTTPLogging struct {
	logger *zap.Logger
}

func NewHTTPLogging(logger *zap.Logger) *HTTPLogging {
	return &HTTPLogging{
		logger: logger,
	}
}

type enrichedResponseWriter struct {
	http.ResponseWriter

	ResponseStatus int
	ResponseSize   int
}

func newEnrichedResponseWriter(w http.ResponseWriter) *enrichedResponseWriter {
	return &enrichedResponseWriter{
		ResponseWriter: w,
	}
}

func (ew *enrichedResponseWriter) WriteHeader(status int) {
	ew.ResponseStatus = status
	ew.ResponseWriter.WriteHeader(status)
}

func (ew *enrichedResponseWriter) Write(b []byte) (int, error) {
	written, err := ew.ResponseWriter.Write(b)
	ew.ResponseSize += written
	return ew.ResponseSize, err
}

func (l *HTTPLogging) Middleware(next http.Handler) http.Handler {
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
			"status", ew.ResponseStatus,
			"responseSize", ew.ResponseSize,
			"durationMs", duration.Milliseconds())
	})
}
