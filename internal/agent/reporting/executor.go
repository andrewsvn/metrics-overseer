package reporting

import (
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

// Executor is responsible for sending given array of metric values to the server
// returns result of two lists: SuccessIDs contains IDs of metrics sent to server
// and FailureIDs contains IDs of those not sent due to errors
type Executor interface {

	// Execute is called in goroutine with prepared metric values
	Execute(ms []*model.Metrics) *Result

	// Shutdown must be called to correctly release all used resources
	Shutdown()
}
