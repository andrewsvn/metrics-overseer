package servercfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var config string = `{
  "database_dsn": "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
  "server_grace_period": 15,
  "key": "abbaabbaupdownselectstart",
  "pprof_address": ":6060",
  "file_storage_path": "storage.path",
  "store_interval": 100,
  "restore": true,
  "crypto_key": "path/to/crypto_key",
  "audit_file": "audit.file",
  "audit_url": "audit.url",
  "pg_max_retry_count": 10,
  "pg_initial_retry_delay_sec": 15,
  "pg_retry_delay_increment_sec": 5
}`

func TestJSONAndDefaultConfigs(t *testing.T) {
	tmpPath := prepareConfigFile(t)

	initialConfig := &Config{}
	jsonConfig, err := NewConfigFromJSONFile(tmpPath)
	require.NoError(t, err)

	initialConfig.FillOutEmptyValues(jsonConfig)
	assert.Equal(t, "", initialConfig.Addr)
	assert.Equal(t, "", initialConfig.LogLevel)
	assert.Equal(t, ":6060", initialConfig.PprofAddr)
	assert.Equal(t, 15, initialConfig.GracePeriodSec)
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable", initialConfig.DBConnString)
	assert.Equal(t, "abbaabbaupdownselectstart", initialConfig.SecretKey)
	assert.Equal(t, "path/to/crypto_key", initialConfig.PrivateKeyPath)
	assert.Equal(t, 100, initialConfig.StoreIntervalSec)
	assert.Equal(t, "storage.path", initialConfig.StorageFilePath)
	assert.Equal(t, true, initialConfig.RestoreOnStartup)
	assert.Equal(t, "audit.file", initialConfig.AuditFilePath)
	assert.Equal(t, 0, initialConfig.AuditFileWriteIntervalSec)
	assert.Equal(t, "audit.url", initialConfig.AuditURL)
	assert.Equal(t, 10, initialConfig.MaxRetryCount)
	assert.Equal(t, 15, initialConfig.InitialRetryDelaySec)
	assert.Equal(t, 5, initialConfig.RetryDelayIncrementSec)

	initialConfig = &Config{
		Addr: ":10000",
		SecurityConfig: SecurityConfig{
			SecretKey: "secretkey",
		},
		FileStorageConfig: FileStorageConfig{
			StorageFilePath: "/tmp/storage",
		},
		AuditConfig: AuditConfig{
			AuditFileWriteIntervalSec: 20,
		},
	}
	initialConfig.FillOutEmptyValues(jsonConfig)
	assert.Equal(t, ":10000", initialConfig.Addr)
	assert.Equal(t, "", initialConfig.LogLevel)
	assert.Equal(t, ":6060", initialConfig.PprofAddr)
	assert.Equal(t, 15, initialConfig.GracePeriodSec)
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable", initialConfig.DBConnString)
	assert.Equal(t, "secretkey", initialConfig.SecretKey)
	assert.Equal(t, "path/to/crypto_key", initialConfig.PrivateKeyPath)
	assert.Equal(t, 100, initialConfig.StoreIntervalSec)
	assert.Equal(t, "/tmp/storage", initialConfig.StorageFilePath)
	assert.Equal(t, true, initialConfig.RestoreOnStartup)
	assert.Equal(t, "audit.file", initialConfig.AuditFilePath)
	assert.Equal(t, 20, initialConfig.AuditFileWriteIntervalSec)
	assert.Equal(t, "audit.url", initialConfig.AuditURL)
	assert.Equal(t, 10, initialConfig.MaxRetryCount)
	assert.Equal(t, 15, initialConfig.InitialRetryDelaySec)
	assert.Equal(t, 5, initialConfig.RetryDelayIncrementSec)
}

func prepareConfigFile(t *testing.T) string {
	tmpPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}
	return tmpPath
}
