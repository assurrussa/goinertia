package goinertia

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// Middleware function.
func (i *Inertia) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if i.csrfTokenProvider != nil && i.csrfTokenCheckProvider != nil && i.isMethodPost(method) {
			if err := i.csrfTokenCheckProvider(c); err != nil {
				return i.redirectCheck(c, err)
			}
		}

		if c.Get(HeaderInertia) == "" {
			err := c.Next()

			return i.redirectCheck(c, err)
		}

		// Check asset version for GET requests only
		if method == http.MethodGet && c.Get(HeaderVersion) != i.assetVersion && !i.isPrecognitionRequest(c) {
			c.Set(HeaderLocation, i.baseURL+c.OriginalURL())
			return c.SendStatus(fiber.StatusConflict)
		}

		// Process the request
		err := c.Next()

		return i.redirectCheck(c, err)
	}
}

func (i *Inertia) MiddlewareErrorListener() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		isAllowedErrorDetailsMessage := i.canExposeDetails(c, c.GetHeaders())
		errReturn := getError(isAllowedErrorDetailsMessage, err, i.customErrorGettingHandler)
		if i.isPrecognitionRequest(c) {
			return i.renderPrecognitionError(c, errReturn)
		}
		details := i.customErrorDetailsHandler(errReturn, isAllowedErrorDetailsMessage)

		if c.Get(HeaderInertia) == "" && c.Method() == fiber.MethodGet {
			return i.renderHTMLError(c, errReturn, details)
		}

		i.WithValidationErrors(c, errReturn.ValidationErrors())
		i.WithFlashMessages(c, errReturn.FlashErrors()...)
		if len(errReturn.ValidationErrors()) == 0 && details != "" {
			i.WithFlashError(c, details)
		}
		err = i.RedirectBack(c)

		return i.redirectCheck(c, err)
	}
}

func (i *Inertia) redirectCheck(c fiber.Ctx, err error) error {
	i.setFlashSessionData(c)

	if c.Get(HeaderInertia) == "" {
		return err
	}

	addVaryHeader(c, HeaderInertia)
	if i.precognitionVary {
		addVaryHeader(c, HeaderPrecognition)
	}
	if i.shouldNoCacheResponse(c) {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}

	method := c.Method()
	statusCode := c.Response().StatusCode()
	if i.isMethodPost(method) && i.isRedirectStatus(statusCode) {
		c.Status(fiber.StatusSeeOther)
		c.Response().SetBodyString("")
	}

	return err
}

func getError(isAllowedErrorDetailsMessage bool, err error, fnGetError func(err error) *Error) *Error {
	if err == nil {
		// fallback error
		return ErrNillable
	}

	if errors.Is(err, ErrInvalidContextViewData) {
		return NewError(fiber.StatusInternalServerError, err.Error(), err)
	}

	if errHTTP := new(Error); errors.As(err, &errHTTP) {
		return errHTTP
	}

	if errHTTP := new(ValidationError); errors.As(err, &errHTTP) {
		return NewError(errHTTP.code, errHTTP.message).CloneValidationError(errHTTP).
			WithFlashErrors(NewFlashError(FlashLevelWarning, errHTTP.message))
	}

	if fnGetError != nil {
		if errCallback := fnGetError(err); errCallback != nil {
			return errCallback
		}
	}

	if errHTTP := new(fiber.Error); errors.As(err, &errHTTP) {
		return NewError(errHTTP.Code, errHTTP.Message, err)
	}

	if isAllowedErrorDetailsMessage {
		return NewError(fiber.StatusInternalServerError, err.Error(), err)
	}

	return ErrInternal
}

func (i *Inertia) isMethodPost(method string) bool {
	return method == http.MethodPost ||
		method == http.MethodPut ||
		method == http.MethodPatch ||
		method == http.MethodDelete
}

func (i *Inertia) isRedirectStatus(statusCode int) bool {
	return statusCode == fiber.StatusFound ||
		statusCode == fiber.StatusMovedPermanently
}
