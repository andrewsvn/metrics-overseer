package audit

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

type FileWriter struct {
	buf      []*Payload
	bufMutex *sync.Mutex
	timer    *time.Ticker

	filename string
	logger   *zap.SugaredLogger
}

func NewFileWriter(filename string, writeIntervalSec int, l *zap.Logger) *FileWriter {
	fw := &FileWriter{
		buf:      make([]*Payload, 0),
		bufMutex: &sync.Mutex{},
		timer:    time.NewTicker(time.Duration(writeIntervalSec) * time.Second),

		filename: filename,
		logger:   l.Sugar().With(zap.String("component", "audit-filewriter")),
	}

	fw.logger.Debugw("starting file auditor", "filename", filename)
	go fw.writeBufferLoop()
	return fw
}

func (fw *FileWriter) OnMetricsUpdate(ts time.Time, ipAddr string, metrics ...*model.Metrics) error {
	fw.bufMutex.Lock()
	defer fw.bufMutex.Unlock()

	// actual file write is done by timer
	payload := NewPayload(ts, ipAddr, metrics...)
	fw.buf = append(fw.buf, payload)
	return nil
}

func (fw *FileWriter) Close() {
	fw.logger.Debugw("closing file auditor", "filename", fw.filename)
	fw.timer.Stop()
	fw.flush()
}

func (fw *FileWriter) writeBufferLoop() {
	for {
		<-fw.timer.C
		fw.flush()
	}
}

func (fw *FileWriter) flush() {
	fw.bufMutex.Lock()
	defer fw.bufMutex.Unlock()

	fw.logger.Debugw("flushing metrics to file",
		zap.String("filename", fw.filename),
		zap.Int("payloadCount", len(fw.buf)),
	)

	if len(fw.buf) == 0 {
		return
	}

	chunkBuilder := strings.Builder{}
	for _, payload := range fw.buf {
		b, err := json.Marshal(payload)
		if err != nil {
			fw.logger.Errorw("error serializing audit payload", zap.Error(err))
			return
		}
		chunkBuilder.Write(b)
		chunkBuilder.WriteString("\n")
	}

	f, err := os.OpenFile(fw.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fw.logger.Errorw("error writing audit payload buffer to file", "filename", fw.filename, "error", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fw.logger.Errorw("error closing file", "filename", fw.filename, "error", err)
		}
	}()

	_, err = f.Write([]byte(chunkBuilder.String()))
	if err != nil {
		fw.logger.Errorw("error writing audit payload buffer to file", "filename", fw.filename, "error", err)
	}

	fw.buf = fw.buf[:0]
}
