package cosmos

import (
	"errors"
	"fmt"
)

// Application error codes.
const (
	EINVALID        = "invalid"
	EINTERNAL       = "internal"
	ECONFLICT       = "conflict"
	ENOTFOUND       = "not_found"
	ENOTIMPLEMENTED = "not_implemented"
)

// Error represents an application-specific error.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("cosmos error: code=%s message=%s", e.Code, e.Message)
}

// ErrorCode unwraps an application error and returns its code.
// Non-application errors always return EINTERNAL to avoid leaking
// sensitive implementation details to the end user.
func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

// ErrorMessage unwraps an application error and returns its message.
// Non-application errors always return "Internal error" to avoid leaking
// sensitive implementation details to the end user.
func ErrorMessage(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error"
}

// Errorf is a helper function to create an application-specific error.
func Errorf(code string, format string, args ...interface{}) *Error {
	// Do not create internal errors. Any error that is not application-specific
	// is, by default, an internal error.
	if code == EINTERNAL {
		panic("internal errors are not application-specific errors")
	}

	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
