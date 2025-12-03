package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/handling/restsrv/errorhandling"
	"go.uber.org/zap"
)

type Decryption struct {
	decrypter encrypt.Decrypter
	logger    *zap.SugaredLogger
}

func NewDecryption(l *zap.Logger, dec encrypt.Decrypter) *Decryption {
	return &Decryption{
		decrypter: dec,
		logger:    l.Sugar().With("component", "decryption-middleware"),
	}
}

func (dec *Decryption) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !dec.decrypter.DecryptingEnabled() {
			next.ServeHTTP(rw, r)
			return
		}

		if r.Body == nil {
			next.ServeHTTP(rw, r)
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			dec.logger.Warnw("can't read incoming request body", "error", err)
			errorhandling.NewValidationHandlerError("can't read incoming request body").Render(rw)
			return
		}

		decrypted, err := dec.decrypter.Decrypt(payload)
		if err != nil {
			dec.logger.Warnw("can't decrypt incoming request body", "error", err)
			errorhandling.NewValidationHandlerError("can't decrypt incoming request body").Render(rw)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(decrypted))
		next.ServeHTTP(rw, r)
	})
}
