package handler

import "net/http"

type HandlerError struct {
	StatusCode int
	Message    string
}

var (
	InternalError *HandlerError = &HandlerError{
		StatusCode: http.StatusInternalServerError,
		Message:    http.StatusText(http.StatusInternalServerError),
	}
)

func NewValidationHandlerError(message string) *HandlerError {
	return &HandlerError{
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func NewNotFoundHandlerError(message string) *HandlerError {
	return &HandlerError{
		StatusCode: http.StatusNotFound,
		Message:    message,
	}
}

func (err *HandlerError) Render(rw http.ResponseWriter) {
	http.Error(rw, err.Message, err.StatusCode)
}
