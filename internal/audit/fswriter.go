package audit

import (
	"encoding/json"
	"os"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

type FileWriter struct {
	filename string
	logger   *zap.SugaredLogger
}

func NewFileWriter(filename string, l *zap.Logger) *FileWriter {
	return &FileWriter{
		filename: filename,
		logger:   l.Sugar().With(zap.String("component", "audit-filewriter")),
	}
}

func (fw FileWriter) OnMetricsUpdate(ts time.Time, ipAddr string, metrics ...*model.Metrics) {
	payload := NewPayload(ts, ipAddr, metrics...)
	b, err := json.Marshal(payload)
	if err != nil {
		fw.logger.Errorw("error serializing payload", "error", err)
		return
	}

	f, err := os.OpenFile(fw.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fw.logger.Errorw("error writing payload to file", "filename", fw.filename, "error", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fw.logger.Errorw("error closing file", "filename", fw.filename, "error", err)
		}
	}()

	_, err = f.Write(append(b, byte('\n')))
	if err != nil {
		fw.logger.Errorw("error writing payload to file", "filename", fw.filename, "error", err)
	}
}
