package retrying

import (
	"errors"
	"go.uber.org/zap"
)

func defaultRetryablePredicate(err error) bool {
	var rerr RetryableError
	return errors.As(err, &rerr)
}

type ExecutorBuilder struct {
	e *Executor
}

func NewExecutorBuilder(p Policy) *ExecutorBuilder {
	return &ExecutorBuilder{
		e: &Executor{
			policy:               p,
			isRetryablePredicate: defaultRetryablePredicate,
		},
	}
}

func (b *ExecutorBuilder) Executor() *Executor {
	return b.e
}

func (b *ExecutorBuilder) WithLogger(logger *zap.SugaredLogger, prefix string) *ExecutorBuilder {
	b.e.logger = logger
	b.e.logPrefix = prefix
	return b
}

func (b *ExecutorBuilder) WithRetryablePredicate(predicate func(err error) bool) *ExecutorBuilder {
	b.e.isRetryablePredicate = predicate
	return b
}
