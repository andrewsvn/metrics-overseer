package retrying

import "time"

type Policy interface {
	MaxAttempts() int
	NextDelay(lastDelay time.Duration) time.Duration
}

// NoRetryPolicy doesn't allow retrying
type NoRetryPolicy struct{}

func (*NoRetryPolicy) MaxAttempts() int {
	return 0
}

func (*NoRetryPolicy) NextDelay(_ time.Duration) time.Duration {
	return 0
}

// LinearPolicy performs retries up to max number, starting with initialDelay interval between reties and increasing it
// linearly by a fixed duration
type LinearPolicy struct {
	maxRetries    int
	initialDelay  time.Duration
	delayIncrease time.Duration
}

func NewLinearPolicy(retries int, delay time.Duration, increase time.Duration) *LinearPolicy {
	return &LinearPolicy{
		maxRetries:    retries,
		initialDelay:  delay,
		delayIncrease: increase,
	}
}

func (p *LinearPolicy) MaxAttempts() int {
	return p.maxRetries
}

func (p *LinearPolicy) NextDelay(lastDelay time.Duration) time.Duration {
	if lastDelay == 0 {
		return p.initialDelay
	}
	return lastDelay + p.delayIncrease
}
