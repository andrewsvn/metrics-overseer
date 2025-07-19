package compress

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

type Compressor struct {
	engines []WriteEngine
	logger  *zap.Logger
}

func NewCompressor(l *zap.Logger, engines ...WriteEngine) *Compressor {
	return &Compressor{
		engines: engines,
		logger:  l,
	}
}

func (c *Compressor) CreateCompressWriter(w http.ResponseWriter, r *http.Request) (CompressedResponseWriter, error) {
	for _, wEngine := range c.engines {
		if wEngine.Applicable(r.Header) {
			c.logger.Debug("Compression engine chosen for response writing",
				zap.String("name", wEngine.Name()))
			crw, err := wEngine.NewResponseWriter(w, 0)
			if err != nil {
				c.logger.Error("Error creating compress writer", zap.Error(err))
				return nil, fmt.Errorf("error creating compress writer: %w", err)
			}
			return crw, nil
		}
	}
	return nil, nil
}
