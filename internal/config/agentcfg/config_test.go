package agentcfg

import (
	"os"
	"path/filepath"
	"testing"

	"dario.cat/mergo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var config string = `{
"report_max_retry_count": 5,
"report_initial_retry_delay_sec": 2,
"rate_limit": 10,
"address": "localhost:7070",
"poll_interval_sec": 1,
"report_interval_sec": 6,
"grace_period_sec": 20,
"crypto_key": "path/to/public_key"
}`

func TestJSONAndDefaultConfigs(t *testing.T) {
	tmpPath := prepareConfigFile(t)

	jsonConfig, err := NewConfigFromJSONFile(tmpPath)
	require.NoError(t, err)

	assert.Equal(t, "localhost:7070", jsonConfig.ServerAddr)
	assert.Equal(t, "", jsonConfig.LogLevel)
	assert.Equal(t, 20, jsonConfig.GracePeriodSec)
	assert.Equal(t, "", jsonConfig.SecretKey)
	assert.Equal(t, "path/to/public_key", jsonConfig.PublicKeyPath)
	assert.Equal(t, 5, jsonConfig.MaxRetryCount)
	assert.Equal(t, 2, jsonConfig.InitialRetryDelaySec)
	assert.Equal(t, 0, jsonConfig.RetryDelayIncrementSec)
	assert.Equal(t, 1, jsonConfig.PollIntervalSec)
	assert.Equal(t, 6, jsonConfig.ReportIntervalSec)

	initialConfig := &Config{
		ServerAddr:     "localhost:10000",
		SecretKey:      "secretkey",
		GracePeriodSec: 40,
		LogLevel:       "debug",
	}

	err = mergo.Merge(initialConfig, jsonConfig)
	require.NoError(t, err)
	assert.Equal(t, "localhost:10000", initialConfig.ServerAddr)
	assert.Equal(t, "debug", initialConfig.LogLevel)
	assert.Equal(t, 40, initialConfig.GracePeriodSec)
	assert.Equal(t, "secretkey", initialConfig.SecretKey)
	assert.Equal(t, "path/to/public_key", initialConfig.PublicKeyPath)
	assert.Equal(t, 5, initialConfig.MaxRetryCount)
	assert.Equal(t, 2, initialConfig.InitialRetryDelaySec)
	assert.Equal(t, 0, initialConfig.RetryDelayIncrementSec)
	assert.Equal(t, 1, initialConfig.PollIntervalSec)
	assert.Equal(t, 6, initialConfig.ReportIntervalSec)
}

func prepareConfigFile(t *testing.T) string {
	tmpPath := filepath.Join(t.TempDir(), "agentconfig.json")
	if err := os.WriteFile(tmpPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}
	return tmpPath
}
