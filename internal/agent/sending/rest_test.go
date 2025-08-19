package sending

import (
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestSender(t *testing.T) {
	logger, _ := logging.NewZapLogger("info")
	retryPolicy := retrying.NewLinearPolicy(3, 1, 2)

	_, err := NewRestSender("http:localhost:8080", retryPolicy, "", logger)
	require.Error(t, err)

	_, err = NewRestSender("http://localhost:8o8o", retryPolicy, "", logger)
	require.Error(t, err)

	rs, err := NewRestSender("localhost:8080", retryPolicy, "", logger)
	require.NoError(t, err)

	url := rs.composePostMetricByPathURL("cnt1", model.Counter, "10")
	assert.Equal(t, "http://localhost:8080/update/counter/cnt1/10", url)
}
