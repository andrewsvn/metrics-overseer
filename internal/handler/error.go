package handler

import "net/http"

type HandlingError struct {
	StatusCode int
	Message    string
	Error      error
}

func NewValidationHandlerError(message string) *HandlingError {
	return &HandlingError{
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func NewNotFoundHandlerError(message string) *HandlingError {
	return &HandlingError{
		StatusCode: http.StatusNotFound,
		Message:    message,
	}
}

func NewInternalServerError(err error) *HandlingError {
	return &HandlingError{
		StatusCode: http.StatusInternalServerError,
		Message:    http.StatusText(http.StatusInternalServerError),
		Error:      err,
	}
}

func (err *HandlingError) Render(rw http.ResponseWriter) {
	http.Error(rw, err.Message, err.StatusCode)
}
