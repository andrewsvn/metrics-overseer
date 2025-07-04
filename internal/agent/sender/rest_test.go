package sender

import (
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRestSender(t *testing.T) {
	rs := NewRestSender("http://localhost:8080")
	url := rs.composePostMetricURL("cnt1", model.Counter, "10")
	assert.Equal(t, "http://localhost:8080/update/counter/cnt1/10", url)
}
