package dump

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"go.uber.org/zap"
	"os"
)

var (
	ErrStore = errors.New("error storing metrics to file")
	ErrLoad  = errors.New("error loading metrics from file")
)

type StorageDumper struct {
	filename string

	ms     repository.Storage
	logger *zap.SugaredLogger
}

func NewStorageDumper(filename string, storage repository.Storage, logger *zap.Logger) *StorageDumper {
	storageLogger := logger.Sugar().With(zap.String("component", "storage-dumper"))
	return &StorageDumper{
		filename: filename,
		ms:       storage,
		logger:   storageLogger,
	}
}

func (sd *StorageDumper) Store() error {
	sd.logger.Infow("Storing metrics to file",
		"filename", sd.filename,
	)

	metrics, err := sd.ms.GetAllSorted()
	if err != nil {
		return fmt.Errorf("error getting metrics to store: %w", err)
	}
	data, err := sd.serializeMetrics(metrics)
	if err != nil {
		return fmt.Errorf("error serializing metrics to JSON: %w", err)
	}
	err = os.WriteFile(sd.filename, data, 0644)
	if err != nil {
		return fmt.Errorf("%w, reason: %v", ErrLoad, err)
	}

	return nil
}

func (sd *StorageDumper) serializeMetrics(metrics []*model.Metrics) ([]byte, error) {
	data := []byte("[\n")

	for i, metric := range metrics {
		bytes, err := json.Marshal(metric)
		if err != nil {
			return nil, err
		}

		data = append(data, []byte("  ")...)
		data = append(data, bytes...)
		if i != len(metrics)-1 {
			data = append(data, ',')
		}
		data = append(data, '\n')
	}
	data = append(data, ']')

	return data, nil
}

func (sd *StorageDumper) Load() error {
	sd.logger.Infow("Loading metrics from file",
		"filename", sd.filename,
	)

	bytes, err := os.ReadFile(sd.filename)
	if err != nil {
		return fmt.Errorf("%w, reason: %v", ErrLoad, err)
	}
	metrics := make([]*model.Metrics, 0)
	err = json.Unmarshal(bytes, &metrics)
	if err != nil {
		return fmt.Errorf("error unmarshalling metrics: %w", err)
	}
	err = sd.ms.SetAll(metrics)
	if err != nil {
		return fmt.Errorf("error storing metrics: %w", err)
	}

	return nil
}
