package middleware

import (
	"github.com/andrewsvn/metrics-overseer/internal/compress"
	"go.uber.org/zap"
	"net/http"
)

type Compressing struct {
	compr  *compress.Compressor
	logger *zap.SugaredLogger
}

func NewCompressing(l *zap.Logger) *Compressing {
	cmLogger := l.Sugar().With(zap.String("component", "compress-middleware"))
	return &Compressing{
		compr:  compress.NewCompressor(l, compress.NewGzipWriteEngine()),
		logger: cmLogger,
	}
}

func (c *Compressing) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
