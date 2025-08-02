package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
	"os"
	"time"
)

var (
	ErrStore = errors.New("error storing metrics to file")
	ErrLoad  = errors.New("error loading metrics from file")
)

type FileStorage struct {
	*MemStorage
	filename    string
	synchronous bool
	logger      *zap.SugaredLogger
}

func NewFileStorage(cfg *servercfg.FileStorageConfig, logger *zap.Logger) *FileStorage {
	fstLogger := logger.Sugar().With(zap.String("component", "file-storage"))
	fst := &FileStorage{
		MemStorage: NewMemStorage(),
		filename:   cfg.StorageFilePath,
		logger:     fstLogger,
	}

	if cfg.RestoreOnStartup {
		err := fst.load()
		if err != nil {
			// this error should be encapsulated here since it doesn't affect the main flow
			logger.Error("failed to load metrics on startup", zap.Error(err))
		}
	}

	if cfg.StoreIntervalSec == 0 {
		fst.synchronous = true
	} else {
		// subscribing on a store timer
		storeInterval := time.Duration(cfg.StoreIntervalSec) * time.Second
		storeTicker := time.NewTicker(storeInterval)
		logger.Info("scheduling metrics storing to file", zap.Duration("interval", storeInterval))
		go func() {
			for {
				<-storeTicker.C
				err := fst.store()
				if err != nil {
					logger.Error("failed to store metrics", zap.Error(err))
				}
			}
		}()
	}

	return fst
}

func (fst *FileStorage) Close() error {
	err := fst.store()
	if err != nil {
		return fmt.Errorf("failed to store metrics on closing: %w", err)
	}
	return nil
}

func (fst *FileStorage) AddCounter(id string, value int64) error {
	err := fst.MemStorage.AddCounter(id, value)
	if err != nil {
		return err
	}
	if fst.synchronous {
		err := fst.store()
		if err != nil {
			return fmt.Errorf("%w, reason: %s", ErrStore, err.Error())
		}
	}
	return nil
}

func (fst *FileStorage) SetGauge(id string, value float64) error {
	err := fst.MemStorage.SetGauge(id, value)
	if err != nil {
		return err
	}
	if fst.synchronous {
		err := fst.store()
		if err != nil {
			return fmt.Errorf("%w, reason: %s", ErrStore, err.Error())
		}
	}
	return nil
}

func (fst *FileStorage) load() error {
	fst.logger.Infow("Loading metrics from file",
		"filename", fst.filename,
	)

	bytes, err := os.ReadFile(fst.filename)
	if err != nil {
		return fmt.Errorf("%w, reason: %v", ErrLoad, err)
	}
	metrics := make([]*model.Metrics, 0)
	err = json.Unmarshal(bytes, &metrics)
	if err != nil {
		return fmt.Errorf("error unmarshalling metrics: %w", err)
	}
	err = fst.SetAll(metrics)
	if err != nil {
		return fmt.Errorf("error storing metrics: %w", err)
	}

	return nil
}

func (fst *FileStorage) store() error {
	fst.logger.Infow("Storing metrics to file",
		"filename", fst.filename,
	)

	metrics, err := fst.GetAllSorted()
	if err != nil {
		return fmt.Errorf("error getting metrics to store: %w", err)
	}
	data, err := fst.serializeMetrics(metrics)
	if err != nil {
		return fmt.Errorf("error serializing metrics to JSON: %w", err)
	}
	err = os.WriteFile(fst.filename, data, 0644)
	if err != nil {
		return fmt.Errorf("%w, reason: %v", ErrLoad, err)
	}

	return nil
}

func (fst *FileStorage) serializeMetrics(metrics []*model.Metrics) ([]byte, error) {
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
