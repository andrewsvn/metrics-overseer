package sender

import (
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestSender(t *testing.T) {
	_, err := NewRestSender("http:localhost:8080")
	require.Error(t, err)

	_, err = NewRestSender("http://localhost:8o8o")
	require.Error(t, err)

	rs, err := NewRestSender("localhost:8080")
	require.NoError(t, err)

	url := rs.composePostMetricURL("cnt1", model.Counter, "10")
	assert.Equal(t, "http://localhost:8080/update/counter/cnt1/10", url)
}
