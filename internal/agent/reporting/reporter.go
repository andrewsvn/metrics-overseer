package reporting

import (
	"context"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

// Reporter is responsible for sending given array of metric values to the server
// returns result of two lists: SuccessIDs contains IDs of metrics sent to server
// and FailureIDs contains IDs of those not sent due to errors
type Reporter interface {
	Execute(ctx context.Context, sndr sender.MetricSender, ms []*model.Metrics) *Result
}
