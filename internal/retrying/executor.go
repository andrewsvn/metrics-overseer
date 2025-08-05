package retrying

import (
	"errors"
	"go.uber.org/zap"
	"time"
)

type Executor struct {
	policy               Policy
	logger               *zap.SugaredLogger
	logPrefix            string
	isRetryablePredicate func(err error) bool
}

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
