package middleware

import (
	"bytes"
	"errors"
	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/handler/errorhandling"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Authorization struct {
	secretKey []byte
	logger    *zap.SugaredLogger
}

func NewAuthorization(l *zap.Logger, secretKey string) *Authorization {
	authLogger := l.Sugar().With("component", "authorization-middleware")
	return &Authorization{
		secretKey: []byte(secretKey),
		logger:    authLogger,
	}
}

func (auth *Authorization) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if auth.secretKey == nil {
			next.ServeHTTP(rw, r)
			return
		}

		var err error

		// backward compatibility: don't verify signature if it's not present in the request even if key is specified
		reqSign, err := encrypt.GetSignature(r.Header)
		if err != nil {
			auth.logger.Warnw("Error getting signature", "error", err)
			errorhandling.NewValidationHandlerError("can't read signature from incoming request").Render(rw)
			return
		}
		if reqSign == nil {
			next.ServeHTTP(rw, r)
			return
		}

		var payload []byte
		if r.Body != nil {
			payload, err = io.ReadAll(r.Body)
			if err != nil {
				auth.logger.Warnw("can't read incoming request body for verification", "error", err)
				errorhandling.NewValidationHandlerError("can't read incoming request body").Render(rw)
				return
			}
		}

		err = encrypt.CheckSignature(auth.secretKey, payload, reqSign)
		if err != nil {
			if errors.Is(err, encrypt.ErrSignatureInvalid) {
				auth.logger.Debugw("incoming request signature is invalid")
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			auth.logger.Warnw("can't verify signature", "error", err)
			errorhandling.NewInternalServerError(err).Render(rw)
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(payload))
		next.ServeHTTP(rw, r)
	})
}
