package middleware

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/handler/errorhandling"
	"go.uber.org/zap"
)

type Authorization struct {
	secretKey     []byte
	trustedSubnet *net.IPNet
	logger        *zap.SugaredLogger
}

func NewAuthorization(l *zap.Logger, secretKey string, trustedSubnet *net.IPNet) *Authorization {
	authLogger := l.Sugar().With("component", "authorization-middleware")
	return &Authorization{
		secretKey:     []byte(secretKey),
		trustedSubnet: trustedSubnet,
		logger:        authLogger,
	}
}

func (auth *Authorization) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if auth.secretKey != nil && !auth.verifySignature(rw, r) {
			return
		}
		if auth.trustedSubnet != nil && !auth.verifyClientIP(rw, r) {
			return
		}
		next.ServeHTTP(rw, r)
	})
}

// verifySignature checks signature header to contain client signature and if it's present, checks if it matches
// a sign of a request body with a client secret key passed in parameters.
// Returns true if sign is correct and false in case verification is failed and processing should be stopped.
// In case of failure all corresponding information (headers, body) is written into response and should not be modified.
func (auth *Authorization) verifySignature(rw http.ResponseWriter, r *http.Request) bool {
	var err error

	// backward compatibility: don't verify signature if it's not present in the request even if key is specified
	reqSign, err := encrypt.GetSignature(r.Header)
	if err != nil {
		auth.logger.Warnw("Error getting signature", "error", err)
		errorhandling.NewValidationHandlerError("can't read signature from incoming request").Render(rw)
		return false
	}
	if reqSign == nil {
		return true
	}

	var payload []byte
	if r.Body != nil {
		payload, err = io.ReadAll(r.Body)
		if err != nil {
			auth.logger.Warnw("can't read incoming request body for verification", "error", err)
			errorhandling.NewValidationHandlerError("can't read incoming request body").Render(rw)
			return false
		}
	}

	err = encrypt.CheckSignature(auth.secretKey, payload, reqSign)
	if err != nil {
		if errors.Is(err, encrypt.ErrSignatureInvalid) {
			auth.logger.Debugw("incoming request signature is invalid")
			rw.WriteHeader(http.StatusUnauthorized)
			return false
		}
		auth.logger.Warnw("can't verify signature", "error", err)
		errorhandling.NewInternalServerError(err).Render(rw)
		return false
	}

	// we need to re-write request body for further processing
	r.Body = io.NopCloser(bytes.NewBuffer(payload))
	return true
}

// verifyClientIP checks for X-Real-IP header of incoming request for valid IP address of a client and
// then checks if it belongs to trustedSubnet.
// Returns true if IP is trusted and false otherwise - in this case processing should be stopped.
// In case of failure all corresponding information (headers, body) is written into response and should not be modified.
func (auth *Authorization) verifyClientIP(rw http.ResponseWriter, r *http.Request) bool {
	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		auth.logger.Debugw("X-Real-IP header of an incoming request is missing")
		rw.WriteHeader(http.StatusUnauthorized)
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		auth.logger.Debugw("X-Real-IP header of an incoming request is invalid")
		rw.WriteHeader(http.StatusUnauthorized)
		return false
	}

	subnet := ip.Mask(auth.trustedSubnet.Mask)
	if !subnet.Equal(auth.trustedSubnet.IP) {
		auth.logger.Debugw("X-Real-IP header of an incoming request is untrusted")
		rw.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}
