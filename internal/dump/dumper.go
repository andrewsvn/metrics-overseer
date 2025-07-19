package dump

import (
	"encoding/json"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"go.uber.org/zap"
	"os"
)

type StorageDumper struct {
	filename string

	ms     repository.Storage
	logger *zap.Logger
}

func NewStorageDumper(filename string, storage repository.Storage, logger *zap.Logger) *StorageDumper {
	return &StorageDumper{
		filename: filename,
		ms:       storage,
		logger:   logger,
	}
}

func (sd *StorageDumper) Store() {
	sd.logger.Info("Storing metrics to file", zap.String("filename", sd.filename))
	metrics, err := sd.ms.GetAllSorted()
	if err != nil {
		sd.logger.Error("Error getting metrics to store", zap.Error(err))
		return
	}
	data, err := sd.serializeMetrics(metrics)
	if err != nil {
		sd.logger.Error("Error serializing metrics to JSON", zap.Error(err))
		return
	}
	err = os.WriteFile(sd.filename, data, 0644)
	if err != nil {
		sd.logger.Error("Error storing metrics to file", zap.Error(err))
	}
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

func (sd *StorageDumper) Load() {
	sd.logger.Info("Loading metrics from file", zap.String("filename", sd.filename))
	bytes, err := os.ReadFile(sd.filename)
	if err != nil {
		sd.logger.Info("Unable to load metrics from file", zap.String("filename", sd.filename))
		return
	}
	metrics := make([]*model.Metrics, 0)
	err = json.Unmarshal(bytes, &metrics)
	if err != nil {
		sd.logger.Error("Error unmarshalling metrics", zap.Error(err))
		return
	}
	err = sd.ms.SetAll(metrics)
	if err != nil {
		sd.logger.Error("Error storing metrics", zap.Error(err))
	}
}
