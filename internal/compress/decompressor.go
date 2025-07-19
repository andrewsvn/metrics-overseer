package compress

import (
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Decompressor struct {
	engines []ReadEngine
	logger  *zap.Logger
}

func NewDecompressor(l *zap.Logger, engines ...ReadEngine) *Decompressor {
	return &Decompressor{
		engines: engines,
		logger:  l,
	}
}

func (d *Decompressor) ReadRequestBody(r *http.Request) ([]byte, error) {
	for _, engine := range d.engines {
		if engine.Applicable(r.Header) {
			d.logger.Debug("Decompress reader chosen for request body", zap.String("name", engine.Name()))
			b, err := engine.ReadAll(r.Body)
			return b, err
		}
	}
	b, err := io.ReadAll(r.Body)
	return b, err
}
