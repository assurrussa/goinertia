package fiberadapter

import (
	"context"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// Middleware function.
func (i *Inertia) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		i.applyVary(c)
		method := c.Method()
		if i.csrfTokenCheckProvider != nil && i.isMethodPost(method) {
			if err := i.csrfTokenCheckProvider(c); err != nil {
				return i.redirectCheck(c, err)
			}
		}

		if c.Get(HeaderInertia) == "" {
			err := c.Next()

			return i.redirectCheck(c, err)
		}

		// Check asset version for GET requests only
		if method == http.MethodGet && c.Get(HeaderVersion) != i.AssetVersion() && !i.isPrecognitionRequest(c) {
			c.Set(HeaderLocation, i.ConflictLocation(c.OriginalURL()))
			if i.IsProtocolV3() {
				c.Set(HeaderVersion, i.AssetVersion())
			}
			err := c.SendStatus(fiber.StatusConflict)
			return i.redirectCheck(c, err)
		}

		// Process the request
		err := c.Next()

		return i.redirectCheck(c, err)
	}
}

func (i *Inertia) MiddlewareErrorListener() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var ctx context.Context = c
		if !i.legacy {
			ctx = Context(c)
		}
		isAllowedErrorDetailsMessage := i.canExposeDetails(ctx, c.GetHeaders())
		errReturn := getError(isAllowedErrorDetailsMessage, err, i.customErrorGettingHandler)
		if i.isPrecognitionRequest(c) {
			return i.renderPrecognitionError(c, errReturn)
		}
		details := i.customErrorDetailsHandler(errReturn, isAllowedErrorDetailsMessage)
		if i.IsProtocolV3() && c.Get(HeaderInertia) != "" && len(errReturn.ValidationErrors()) == 0 {
			status := errReturn.Code
			if status < http.StatusBadRequest || status > 599 {
				status = http.StatusInternalServerError
			}
			c.Response().Header.Del(HeaderInertia)
			c.Response().Header.Del(HeaderLocation)
			c.Response().Header.Del(HeaderRedirect)
			c.Response().Header.Del(fiber.HeaderLocation)
			return c.Status(status).JSON(map[string]string{errorMessageKey: details})
		}

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

	i.applyVary(c)
	if i.shouldNoCacheResponse(c) {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}

	method := c.Method()
	statusCode := c.Response().StatusCode()
	location := string(c.Response().Header.Peek(fiber.HeaderLocation))
	if i.IsFragmentRedirect(c.Get(HeaderInertia), isPrefetch(c), statusCode, location) {
		c.Set(HeaderRedirect, location)
		c.Response().Header.Del(fiber.HeaderLocation)
		c.Response().Header.Del(HeaderInertia)
		c.Response().Header.Del(fiber.HeaderContentLength)
		c.Status(fiber.StatusConflict)
		c.Response().SetBodyString("")
		return err
	}
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
		return NewError(errHTTP.StatusCode(), errHTTP.Error()).CloneValidationError(errHTTP).
			WithFlashErrors(NewFlashError(FlashLevelWarning, errHTTP.Error()))
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

func (i *Inertia) isRedirectStatus(statusCode int) bool {
	return statusCode == fiber.StatusFound ||
		statusCode == fiber.StatusMovedPermanently
}
