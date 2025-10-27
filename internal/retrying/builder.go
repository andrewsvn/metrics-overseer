package retrying

import (
	"errors"

	"go.uber.org/zap"
)

func defaultRetryablePredicate(err error) bool {
	var rerr RetryableError
	return errors.As(err, &rerr)
}

// ExecutorBuilder must be used to create Executor instances based on combination of parameters needed:
// - retry policy, should always be provided
// - retryablePredicate - customizable method to determine if execution error is retryable
// - zap logger with optional custom prefix - can be used to log retry attempts in executor
type ExecutorBuilder struct {
	e *Executor
}

// NewExecutorBuilder initializes builder with retry policy as it should be always provided
// it stores an internal instance of Executor which can be then returned by calling ExecutorBuilder.Build
func NewExecutorBuilder(p Policy) *ExecutorBuilder {
	return &ExecutorBuilder{
		e: &Executor{
			policy:               p,
			isRetryablePredicate: defaultRetryablePredicate,
		},
	}
}

// Build returns its current instance of Executor
func (b *ExecutorBuilder) Build() *Executor {
	return b.e
}

// WithLogger adds zap.SugaredLogger with optional string prefix to Executor instance
// If this method is not called, retry attempts will not be logged in Executor
func (b *ExecutorBuilder) WithLogger(logger *zap.SugaredLogger, prefix string) *ExecutorBuilder {
	b.e.logger = logger
	b.e.logPrefix = prefix
	return b
}

// WithRetryablePredicate adds custom predicate function which can be used to determine if error is retryable
// If this method is not called, Executor will use default retryable predicate which checks if error is RetryableError
func (b *ExecutorBuilder) WithRetryablePredicate(predicate func(err error) bool) *ExecutorBuilder {
	b.e.isRetryablePredicate = predicate
	return b
}
