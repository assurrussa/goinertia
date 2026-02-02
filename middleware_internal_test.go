package goinertia

import (
	"errors"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

type testServerError struct {
	code    int
	message string
	cause   error
}

func (e *testServerError) Error() string { return e.message }
func (e *testServerError) Unwrap() error { return e.cause }

type testFieldError struct {
	Field string
}

type testValidationError struct {
	code    int
	message string
	fields  []testFieldError
	cause   error
}

func (e *testValidationError) Error() string { return e.message }
func (e *testValidationError) Unwrap() error { return e.cause }

func Test_getError(t *testing.T) {
	t.Parallel()

	fnCall := func(appErr *Error) func(checkErr error) *Error {
		return func(_ error) *Error {
			return appErr
		}
	}

	fnServerValidationError := func(checkErr error) *Error {
		var e *testValidationError
		if errors.As(checkErr, &e) {
			errHTTP := NewError(e.code, e.message, e.Unwrap())
			if len(e.fields) == 0 {
				return errHTTP
			}

			vals := make(ValidationErrors, len(e.fields))
			for _, field := range e.fields {
				vals[field.Field] = []string{field.Field}
			}

			return errHTTP.CloneValidationError(NewValidationError(errHTTP.Code, errHTTP.Message, vals)).
				WithFlashErrors(NewFlashError(FlashLevelWarning, errHTTP.Message))
		}

		return nil
	}

	fnServerError := func(checkErr error) *Error {
		var e *testServerError
		if errors.As(checkErr, &e) {
			return NewError(e.code, e.message, e.Unwrap())
		}

		return nil
	}

	fnAppValidationError := func(checkErr error) *Error {
		var e *ValidationError
		if errors.As(checkErr, &e) {
			return NewError(e.code, e.message).CloneValidationError(e).
				WithFlashErrors(NewFlashError(FlashLevelWarning, e.message))
		}

		return nil
	}

	errApp := NewError(
		419,
		"test error",
		errors.New("test2 error"),
		errors.New("test3 error"),
	)

	errAppValidationErrors := NewValidationError(400, "Bad request", ValidationErrors{
		"password": {"Не валидный пароль"},
		"name":     {"Не валидное имя"},
	})
	callbackErr := errors.New("callback error")
	callbackReturn := NewError(418, "callback handled", callbackErr)
	callbackOverride := NewError(402, "callback override", errors.New("override"))

	tests := []struct {
		isAllowedDetails bool
		checkErr         error
		callbackErr      func(err error) *Error
		expectErr        func(checkErr error) *Error
	}{
		{isAllowedDetails: false, checkErr: nil, expectErr: fnCall(ErrNillable)},
		{isAllowedDetails: true, checkErr: nil, expectErr: fnCall(ErrNillable)},
		{
			isAllowedDetails: false,
			checkErr:         ErrInvalidContextViewData,
			expectErr: func(checkErr error) *Error {
				return NewError(fiber.StatusInternalServerError, checkErr.Error(), checkErr)
			},
		},
		{
			isAllowedDetails: true,
			checkErr:         ErrInvalidContextViewData,
			expectErr: func(checkErr error) *Error {
				return NewError(fiber.StatusInternalServerError, checkErr.Error(), checkErr)
			},
		},
		{
			isAllowedDetails: false,
			checkErr:         errApp,
			expectErr:        fnCall(errApp),
		},
		{
			isAllowedDetails: true,
			checkErr:         errApp,
			expectErr:        fnCall(errApp),
		},
		{
			isAllowedDetails: false,
			checkErr:         errAppValidationErrors,
			expectErr:        fnAppValidationError,
		},
		{
			isAllowedDetails: true,
			checkErr:         errAppValidationErrors,
			expectErr:        fnAppValidationError,
		},
		{
			isAllowedDetails: false,
			checkErr:         errors.New("test error"),
			expectErr:        fnCall(ErrInternal),
		},
		{
			isAllowedDetails: true,
			checkErr:         errors.New("test error"),
			expectErr: func(checkErr error) *Error {
				return NewError(fiber.StatusInternalServerError, checkErr.Error(), checkErr)
			},
		},
		{
			isAllowedDetails: false,
			checkErr:         fiber.NewError(419, "test error"),
			expectErr: func(checkErr error) *Error {
				return NewError(419, "test error", checkErr)
			},
		},
		{
			isAllowedDetails: true,
			checkErr:         fiber.NewError(419, "test error"),
			expectErr: func(checkErr error) *Error {
				return NewError(419, "test error", checkErr)
			},
		},
		{
			isAllowedDetails: false,
			checkErr:         &testServerError{code: 419, message: "test error", cause: errors.New("test2 message")},
			callbackErr:      fnServerError,
			expectErr:        fnServerError,
		},
		{
			isAllowedDetails: true,
			checkErr:         &testServerError{code: 419, message: "test error", cause: errors.New("test2 message")},
			callbackErr:      fnServerError,
			expectErr:        fnServerError,
		},
		{
			isAllowedDetails: false,
			checkErr: &testValidationError{
				code:    400,
				message: "Validation failed",
				fields: []testFieldError{
					{Field: "email"},
				},
				cause: errors.New("test2 message"),
			},
			callbackErr: fnServerValidationError,
			expectErr:   fnServerValidationError,
		},
		{
			isAllowedDetails: true,
			checkErr: &testValidationError{
				code:    400,
				message: "Validation failed",
				fields: []testFieldError{
					{Field: "email"},
				},
				cause: errors.New("test2 message"),
			},
			callbackErr: fnServerValidationError,
			expectErr:   fnServerValidationError,
		},
		{
			isAllowedDetails: false,
			checkErr: &testValidationError{
				code:    400,
				message: "Validation failed",
				fields:  nil,
				cause:   errors.New("test2 message"),
			},
			callbackErr: fnServerValidationError,
			expectErr:   fnServerValidationError,
		},
		{
			isAllowedDetails: true,
			checkErr: &testValidationError{
				code:    400,
				message: "Validation failed",
				fields:  nil,
				cause:   errors.New("test2 message"),
			},
			callbackErr: fnServerValidationError,
			expectErr:   fnServerValidationError,
		},
		{
			isAllowedDetails: false,
			checkErr:         callbackErr,
			callbackErr: func(err error) *Error {
				if errors.Is(err, callbackErr) {
					return callbackReturn
				}
				return nil
			},
			expectErr: fnCall(callbackReturn),
		},
		{
			isAllowedDetails: false,
			checkErr:         callbackErr,
			callbackErr: func(_ error) *Error {
				return nil
			},
			expectErr: fnCall(ErrInternal),
		},
		{
			isAllowedDetails: false,
			checkErr:         fiber.NewError(403, "forbidden"),
			callbackErr: func(_ error) *Error {
				return callbackOverride
			},
			expectErr: fnCall(callbackOverride),
		},
		{
			isAllowedDetails: false,
			checkErr:         errAppValidationErrors,
			callbackErr: func(_ error) *Error {
				return callbackOverride
			},
			expectErr: fnAppValidationError,
		},
	}

	for i, tt := range tests {
		t.Run("case #"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			err := getError(tt.isAllowedDetails, tt.checkErr, tt.callbackErr)
			assert.Equal(t, tt.expectErr(tt.checkErr), err)
		})
	}
}
