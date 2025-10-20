package retrying

import (
	"errors"
	"time"

	"go.uber.org/zap"
)

// Executor provides a wrapper to execute a block of code which can fail with some non-persistent (retriable) error
// It must be initialized using ExecutorBuilder, by default it has no retries and no logging.
type Executor struct {
	policy               Policy
	logger               *zap.SugaredLogger
	logPrefix            string
	isRetryablePredicate func(err error) bool
}

// Run performs one or more executions of provided function f - until it either returns no error or
// no more retries allowed by a retry policy, or error returned is considered as non-retriable.
// To check if an error is retriable, returned error is checked using e.isRetryablePredicate method.
// In case the error is retriable, e.policy checks if next retry is allowed - based on number of previous tries.
// If next retry is allowed, then e.policy determines needed delay before it.
// If e.logger is provided, all retry attempts will be logged.
// Returns error value got from the last f execution
func (e *Executor) Run(f func() error) error {
	var err error
	err = f()
	if err == nil {
		return nil
	}
	if !e.isRetryablePredicate(err) {
		return err
	}

	attempt := 0
	delay := e.policy.NextDelay(0)
	e.logInfo("scheduling retry attempt",
		"reason", err.Error(),
		"delay", delay.String(),
	)

	for attempt < e.policy.MaxAttempts() {
		time.Sleep(delay)

		attempt += 1
		e.logInfo("retry attempt",
			"attempt", attempt,
		)
		err = f()
		if err == nil {
			return nil
		}
		if !e.isRetryablePredicate(err) {
			return err
		}

		delay = e.policy.NextDelay(delay)
		e.logInfo("retry attempt failed, scheduling next retry",
			"reason", err.Error(),
			"delay", delay.String(),
		)
	}

	e.logError("max number of retry attempts reached")
	if errors.Is(err, &RetryableError{}) {
		return errors.Unwrap(err)
	}
	return err
}

func (e *Executor) logInfo(msg string, keysAndValues ...interface{}) {
	if e.logger == nil {
		return
	}
	e.logger.Infow(e.logPrefix+": "+msg, keysAndValues...)
}

func (e *Executor) logError(msg string, keysAndValues ...interface{}) {
	if e.logger == nil {
		return
	}
	e.logger.Errorw(e.logPrefix+": "+msg, keysAndValues...)
}
