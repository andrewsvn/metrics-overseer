package retrying

// RetryableError is a simple way to mark error in retryable code as retryable
// this is an alternative to creating a custom retryable predicate function
type RetryableError struct {
	error
}

func NewRetryableError(err error) RetryableError {
	return RetryableError{err}
}

func (e *RetryableError) Unwrap() error {
	return e.error
}
