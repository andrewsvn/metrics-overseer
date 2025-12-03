package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/compress"
	"github.com/andrewsvn/metrics-overseer/internal/handling/restsrv/errorhandling"
	"go.uber.org/zap"
)

type Compressing struct {
	compr   *compress.Compressor
	decompr *compress.Decompressor
	logger  *zap.SugaredLogger
}

func NewCompressing(l *zap.Logger) *Compressing {
	cmLogger := l.Sugar().With(zap.String("component", "compress-middleware"))
	return &Compressing{
		compr:   compress.NewCompressor(l, compress.NewGzipWriteEngine()),
		decompr: compress.NewDecompressor(l, compress.NewGzipReadEngine()),
		logger:  cmLogger,
	}
}

func (c *Compressing) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := c.decompr.ReadRequestBody(r)
		if err != nil {
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error decompressing body: %v", err)).Render(w)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		crw, err := c.compr.CreateCompressWriter(w, r)
		if err != nil {
			c.logger.Error("error creating compress writer", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if crw != nil {
			defer func() {
				err = crw.Close()
				if err != nil {
					c.logger.Error("Error closing compress writer", zap.Error(err))
				}
			}()
			next.ServeHTTP(crw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
