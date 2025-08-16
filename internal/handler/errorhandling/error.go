package errorhandling

import "net/http"

type Error struct {
	StatusCode int
	Message    string
	Error      error
}

func NewValidationHandlerError(message string) *Error {
	return &Error{
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func NewNotFoundHandlerError(message string) *Error {
	return &Error{
		StatusCode: http.StatusNotFound,
		Message:    message,
	}
}

func NewInternalServerError(err error) *Error {
	return &Error{
		StatusCode: http.StatusInternalServerError,
		Message:    http.StatusText(http.StatusInternalServerError),
		Error:      err,
	}
}

func (err *Error) Render(rw http.ResponseWriter) {
	http.Error(rw, err.Message, err.StatusCode)
}
