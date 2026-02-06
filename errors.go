package goinertia

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/utils/v2"
)

var (
	// ErrInvalidContextViewData error.
	ErrInvalidContextViewData = errors.New("inertia: could not convert context view data to map")
	// ErrBadSsrStatusCode error.
	ErrBadSsrStatusCode = errors.New("inertia: bad processSSR status code >= 400")
	// ErrBaseURLEmpty error.
	ErrBaseURLEmpty = errors.New("base URL is empty")
)

type ValidationErrors map[string][]string

var (
	ErrNillable = NewError(500, "inertia: value is nil")
	ErrInternal = NewError(fiber.StatusInternalServerError, utils.StatusMessage(fiber.StatusInternalServerError))
)

// Error represents an error that occurred while handling a request.
type Error struct {
	Code             int
	Message          string
	cause            error
	flashErrors      []FlashError
	validationErrors ValidationErrors
}

// Error makes it compatible with the `error` interface.
func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.cause
}

// NewError creates a new Error instance with an optional message.
func NewError(code int, message string, errs ...error) *Error {
	err := &Error{
		Code:    code,
		Message: utils.StatusMessage(code),
	}

	if message != "" {
		err.Message = message
	}

	if len(errs) > 0 {
		err.cause = errors.Join(errs...)
	}

	return err
}

func (e *Error) CloneValidationError(err *ValidationError) *Error {
	newErr := NewError(err.code, err.Error(), e.Unwrap())
	newErr.validationErrors = err.Errors()

	return newErr
}

func (e *Error) WithFlashErrors(errs ...*FlashError) *Error {
	if e.flashErrors == nil {
		e.flashErrors = make([]FlashError, 0, len(errs))
	}

	for _, flashError := range errs {
		e.flashErrors = append(e.flashErrors, *flashError)
	}

	return e
}

func (e *Error) FlashErrors() []FlashError {
	return e.flashErrors
}

func (e *Error) ValidationErrors() ValidationErrors {
	return e.validationErrors
}

// FlashLevel is a string enum for flash message level.
type FlashLevel string

const (
	FlashLevelSuccess FlashLevel = "success"
	FlashLevelInfo    FlashLevel = "info"
	FlashLevelWarning FlashLevel = "warning"
	FlashLevelError   FlashLevel = "error"
)

func (f FlashLevel) String() string {
	return string(f)
}

// FlashError is a rich error that carries, UI message level and the user-facing message.
type FlashError struct {
	Level   FlashLevel
	Message string
}

// NewFlashError creates a FlashError.
func NewFlashError(level FlashLevel, userMessage string) *FlashError {
	return &FlashError{Level: level, Message: userMessage}
}

func (e *FlashError) Error() string { return e.Message }

type ValidationError struct {
	code    int
	message string
	errs    ValidationErrors
}

// NewValidationError creates a ValidationError.
func NewValidationError(code int, userMessage string, errs ValidationErrors) *ValidationError {
	return &ValidationError{
		code:    code,
		message: userMessage,
		errs:    errs,
	}
}

func (e *ValidationError) Error() string            { return e.message }
func (e *ValidationError) Errors() ValidationErrors { return e.errs }
