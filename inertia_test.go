package goinertia_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/inertiat"
	inertiamocks "github.com/assurrussa/goinertia/mocks"
)

func TestInertia_WithFS(t *testing.T) {
	t.Parallel()

	testInertia(t, goinertia.WithRootHotTemplate(""))
}

func TestInertia_WithEmpty(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		inertiat.NewTestApp(
			t,
			goinertia.WithRootTemplate(""),
			goinertia.WithRootErrorTemplate(""),
		)
	})
}

func TestInertia_WithoutFS(t *testing.T) {
	t.Parallel()

	testInertia(t, goinertia.WithPublicFS(nil))
}

func TestInertia_WithoutPublicFSAndWithotHotFile(t *testing.T) {
	t.Parallel()

	testInertia(
		t,
		goinertia.WithRootHotTemplate("public/static/hot"),
		goinertia.WithPublicFS(nil),
	)
}

func TestInertia_WithHot_WithPublicFS(t *testing.T) {
	t.Parallel()

	testInertiaHot(t)
}

func TestInertia_WithHot_WithoutPublicFS(t *testing.T) {
	t.Parallel()

	testInertiaHot(
		t,
		goinertia.WithRootHotTemplate("testdata/public/static/hot"),
		goinertia.WithPublicFS(nil),
	)
}

func TestInertia_ParseTemplates(t *testing.T) {
	t.Parallel()

	ta := inertiat.NewTestApp(t)
	require.NoError(t, ta.Inrt.ParseTemplates())

	ta = inertiat.NewTestApp(t, goinertia.WithRootTemplate("errorpath.html"))
	require.Error(t, ta.Inrt.ParseTemplates())

	ta = inertiat.NewTestApp(t, goinertia.WithRootErrorTemplate("errorpath.html"))
	require.Error(t, ta.Inrt.ParseTemplates())
}

func TestInertia_DefaultCanExpose(t *testing.T) {
	t.Parallel()

	ta := inertiat.NewTestApp(t)
	assert.False(t, goinertia.DefaultCanExpose(fiber.NewDefaultCtx(ta.App), nil))
}

func TestInertia_DefaultCustomErrorDetails_IsNotCan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    int
		message string
		details string
	}{
		{code: 9999, message: "Unknown error", details: "Something went wrong. Try again later"},
		{code: fiber.StatusBadRequest, message: "Bad request error", details: "Bad request"},
		{code: fiber.StatusNotFound, message: "Page not found error", details: "Page not found"},
		{code: fiber.StatusForbidden, message: "Permission denied error", details: "Permission denied"},
		{code: fiber.StatusUnauthorized, message: "Unauthorized error", details: "Unauthorized"},
		{code: 419, message: "The page expired, please try again error", details: "The page expired, please try again"},
		{code: fiber.StatusTooManyRequests, message: "Too many request error", details: "Too many request"},
		{
			code: fiber.StatusInternalServerError, message: "Something went wrong. Try again later error",
			details: "Something went wrong. Try again later",
		},
		{
			code: fiber.StatusNotImplemented, message: "Something went wrong. Try again later error",
			details: "Something went wrong. Try again later",
		},
	}

	for i, tt := range tests {
		t.Run("case #"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			err := goinertia.NewError(tt.code, tt.message)
			details := goinertia.DefaultCustomErrorDetails(err, false)
			assert.Equal(t, tt.details, details)
		})
	}
}

func TestInertia_DefaultCustomErrorDetails_IsCan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    int
		message string
		details string
	}{
		{code: 9999, message: "Unknown error", details: "Unknown error"},
		{code: fiber.StatusBadRequest, message: "Bad request error", details: "Bad request error"},
		{code: fiber.StatusNotFound, message: "Page not found error", details: "Page not found error"},
		{code: fiber.StatusForbidden, message: "Permission denied error", details: "Permission denied error"},
		{code: fiber.StatusUnauthorized, message: "Unauthorized error", details: "Unauthorized error"},
		{code: 419, message: "The page expired, please try again error", details: "The page expired, please try again error"},
		{code: fiber.StatusTooManyRequests, message: "Too many request error", details: "Too many request error"},
		{
			code: fiber.StatusInternalServerError, message: "Something went wrong. Try again later error",
			details: "Something went wrong. Try again later error",
		},
		{
			code: fiber.StatusNotImplemented, message: "Something went wrong. Try again later error",
			details: "Something went wrong. Try again later error",
		},
	}

	for i, tt := range tests {
		t.Run("case #"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			err := goinertia.NewError(tt.code, tt.message)
			details := goinertia.DefaultCustomErrorDetails(err, true)
			assert.Equal(t, tt.details, details)
		})
	}
}

func TestInertia_ErrorTemplateFallback(t *testing.T) {
	t.Parallel()

	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithRootErrorTemplate(
		"does/not/exist.gohtml",
	))
	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(_ fiber.Ctx) error { return errors.New("boom") }, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, "Internal server error", body)
}

func TestInertia_WithLazyProp(t *testing.T) {
	callCount := 0
	//nolint:unparam // tests
	expensiveComputation := func(_ context.Context) (any, error) {
		callCount++

		return map[string]any{
			"data":      "expensive computation result",
			"callCount": callCount,
		}, nil
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t)
	handler := func(c fiber.Ctx) error {
		ta.Inrt.WithLazyProp(c, "expensiveData", expensiveComputation)
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}
	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, nil)
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])

	expensiveData, ok := page.Props["expensiveData"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "expensive computation result", expensiveData["data"])
	assert.InEpsilon(t, float64(1), expensiveData["callCount"], 0.1) // JSON numbers are float64

	callCount = 0 // Reset counter

	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(handler, map[string]string{
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	})
	page2 := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "John", page2.Props["user"])
	assert.NotContains(t, page2.Props, "expensiveData")
	assert.Equal(t, 0, callCount) // Should not be called
}

func TestInertia_LazyPropPartialRequest(t *testing.T) {
	callCount := 0
	//nolint:unparam // tests
	expensiveComputation := func(_ context.Context) (any, error) {
		callCount++
		return "expensive result", nil
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t)
	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		ta.Inrt.WithLazyProp(c, "expensiveData", expensiveComputation)
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, map[string]string{
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "expensiveData",
	})
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "expensive result", page.Props["expensiveData"])
	assert.NotContains(t, page.Props, "user") // Should not include other props
	assert.Equal(t, 1, callCount)             // Should be called once
}

func TestInertia_LazyPropError(t *testing.T) {
	failingComputation := func(_ context.Context) (any, error) {
		return nil, assert.AnError
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t)
	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		ta.Inrt.WithLazyProp(c, "failingData", failingComputation)
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "John", page.Props["user"])
	assert.NotContains(t, page.Props, "failingData") // Should not include failing prop
}

func TestInertia_LazyPropShared(t *testing.T) {
	callCount := 0

	expensiveComputation := func(_ context.Context) (any, error) {
		callCount++
		return "shared expensive result", nil
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t, goinertia.WithSharedProps(map[string]any{
		"sharedLazy": goinertia.LazyProp{Key: "sharedLazy", Fn: expensiveComputation},
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, map[string]string{
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	})
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "John", page.Props["user"])
	assert.NotContains(t, page.Props, "sharedLazy")
	assert.Equal(t, 0, callCount) // Should not be called

	// Test full request that includes the shared lazy prop
	callCount = 0
	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)
	page2 := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "John", page2.Props["user"])
	assert.Equal(t, "shared expensive result", page2.Props["sharedLazy"])
	assert.Equal(t, 1, callCount)
}

func TestInertia_LazyPropSharedOverriddenByContext(t *testing.T) {
	sharedCalls := 0
	localCalls := 0

	sharedComputation := func(_ context.Context) (any, error) {
		sharedCalls++
		return "shared value", nil
	}
	//nolint:unparam // it's valid - result 1 (error) is always ni
	localComputation := func(_ context.Context) (any, error) {
		localCalls++
		return "local value", nil
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t, goinertia.WithSharedProps(map[string]any{
		"sharedLazy": goinertia.LazyProp{Key: "sharedLazy", Fn: sharedComputation},
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		ta.Inrt.WithLazyProp(c, "sharedLazy", localComputation)
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "local value", page.Props["sharedLazy"])
	assert.Equal(t, 0, sharedCalls)
	assert.Equal(t, 1, localCalls)
}

func TestInertia_LazyPropSharedOverriddenByRequest(t *testing.T) {
	sharedCalls := 0
	sharedComputation := func(_ context.Context) (any, error) {
		sharedCalls++
		return "shared value", nil
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t, goinertia.WithSharedProps(map[string]any{
		"sharedLazy": goinertia.LazyProp{Key: "sharedLazy", Fn: sharedComputation},
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user":       "John",
			"sharedLazy": "request value",
		})
	}, nil)
	page := inertiat.DecodePage(t, body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "request value", page.Props["sharedLazy"])
	assert.Equal(t, 0, sharedCalls)
}

func testInertia(t *testing.T, opts ...goinertia.Option) {
	t.Helper()

	ta := inertiat.NewTestApp(t, opts...)

	require.NoError(t, ta.Inrt.ParseTemplates())

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", fiber.Map{
			"message": "Hello from Fiber!",
		})
	}, nil)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, body, `<script src="/public/dist/js/app.js" defer></script>`)
	assert.Contains(t, body, `<link href="/public/dist/css/app.css" rel="stylesheet">`)
	assert.Contains(t, body, `Hello from Fiber!`)
	assert.Contains(t, body, `embed-test123/test-text.html`)
}

func testInertiaHot(t *testing.T, opts ...goinertia.Option) {
	t.Helper()

	ta := inertiat.NewTestApp(t, opts...)

	require.NoError(t, ta.Inrt.ParseTemplates())

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", fiber.Map{
			"message": "Hello from Fiber!",
		})
	}, nil)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, body, `<script type="module" src="/@vite/client"></script>`)
	assert.Contains(t, body, `<script type="module" src="/src/js/app.js"></script>`)
	assert.NotContains(t, body, `<link href="/css/app.css" rel="stylesheet">`)
	assert.Contains(t, body, `Hello from Fiber!`)
	assert.Contains(t, body, `embed-test123/test-text.html`)
}

func TestInertia_ValidationErrorMapping_GetError(t *testing.T) {
	ta := inertiat.NewTestAppWithErrorHandler(t)
	errAppValidationErrors := goinertia.NewValidationError(400, "Bad request", goinertia.ValidationErrors{
		"password": {"Не валидный пароль"},
		"name":     {"Не валидное имя"},
	})
	handler := func(_ fiber.Ctx) error { return errAppValidationErrors }

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(handler, nil)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, `<div class="error-code">400</div>
    <div class="error-message">Bad request</div>
    
    <div class="error-details">
        <h3>Details:</h3>
        <pre>Bad request</pre>
    </div>`)

	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, nil)
	assert.Equal(t, fiber.StatusFound, resp.StatusCode)
	assert.Empty(t, body)
}

func TestInertia_CanExposeDetailsCallback(t *testing.T) {
	ta := inertiat.NewTestAppWithErrorHandler(t,
		goinertia.WithCanExposeDetails(func(_ context.Context, headers map[string][]string) bool {
			values, ok := headers["X-Debug"]
			if !ok {
				return false
			}
			return values[0] == "1"
		}),
	)

	handler := func(_ fiber.Ctx) error { return errors.New("detailed failure") }
	//nolint:bodyclose // tests
	resp, body := ta.DoGet(handler, map[string]string{
		"Referer": "/prev",
	})
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, body, `Internal Server Error`)
	assert.Contains(t, body, `Something went wrong. Try again later`)

	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		"Referer": "/prev",
	})
	assert.Equal(t, fiber.StatusFound, resp.StatusCode)
	assert.Empty(t, body)

	// 2) Non-Inertia with permission: expect HTML containing details
	//nolint:bodyclose // tests
	resp, body = ta.DoGet(handler, map[string]string{
		"Referer": "/prev",
		"X-Debug": "1",
	})
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, body, `detailed failure: detailed failure`)

	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		"Referer": "/prev",
		"X-Debug": "1",
	})
	assert.Equal(t, fiber.StatusFound, resp.StatusCode)
	assert.Empty(t, body)
}

func TestInertia_Redirect(t *testing.T) {
	tests := []struct {
		name           string
		inertiaRequest bool
		url            string
		expectedStatus int
		external       bool
	}{
		{
			name:           "inertia request should return 302 with location header",
			inertiaRequest: true,
			url:            "/dashboard",
			expectedStatus: fiber.StatusFound,
		},
		{
			name:           "inertia request with external URL should return 409 with X-Inertia-Location",
			inertiaRequest: true,
			url:            "https://example.com/dashboard",
			expectedStatus: fiber.StatusConflict,
			external:       true,
		},
		{
			name:           "regular request should return 302 redirect",
			inertiaRequest: false,
			url:            "/dashboard",
			expectedStatus: fiber.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := inertiat.NewTestAppWithoutMiddleware(t)

			headers := map[string]string{}
			if tt.inertiaRequest {
				headers[goinertia.HeaderInertia] = "true"
			}
			//nolint:bodyclose // tests
			resp, _ := ta.DoGet(func(c fiber.Ctx) error {
				if tt.inertiaRequest {
					c.Set(goinertia.HeaderInertia, "true")
				}
				return ta.Inrt.Redirect(c, tt.url)
			}, headers)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.external {
				assert.Equal(t, tt.url, resp.Header.Get(goinertia.HeaderLocation))
				assert.Equal(t, tt.url, resp.Header.Get(fiber.HeaderLocation))
			} else {
				assert.Equal(t, tt.url, resp.Header.Get(fiber.HeaderLocation))
				assert.Empty(t, resp.Header.Get(goinertia.HeaderLocation))
			}
		})
	}
}

func TestInertia_RedirectHelpers_NotXHR(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(c fiber.Ctx) error {
		return goinertia.Redirect(c, "/dashboard")
	}, nil)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, resp.Header.Get(goinertia.HeaderLocation))
	assert.Equal(t, "/dashboard", resp.Header.Get(fiber.HeaderLocation))
	assert.Empty(t, body)
}

func TestInertia_RedirectHelpers_XHR(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return goinertia.Redirect(c, "/dashboard")
	}, nil)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, resp.Header.Get(goinertia.HeaderLocation))
	assert.Equal(t, "/dashboard", resp.Header.Get(fiber.HeaderLocation))
	assert.Empty(t, body)
}

func TestInertia_RedirectHelpers_External(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return goinertia.RedirectExternal(c, "https://example.com/dashboard")
	}, nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "https://example.com/dashboard", resp.Header.Get(goinertia.HeaderLocation))
	assert.Equal(t, "https://example.com/dashboard", resp.Header.Get(fiber.HeaderLocation))
	assert.Equal(t, "Conflict", body)
}

func TestInertia_RedirectHelpers_Empty(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return goinertia.Redirect(c, "")
	}, nil)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, resp.Header.Get(goinertia.HeaderLocation))
	assert.Equal(t, "http://example.com", resp.Header.Get(fiber.HeaderLocation))
	assert.Empty(t, body)

	ta = inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(func(c fiber.Ctx) error {
		return goinertia.Redirect(c, "/")
	}, nil)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, resp.Header.Get(goinertia.HeaderLocation))
	assert.Equal(t, "http://example.com", resp.Header.Get(fiber.HeaderLocation))
	assert.Empty(t, body)
}

func TestInertia_RedirectBackWithErrors(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.RedirectBackWithErrors(c, map[string]string{"field1": "Error 1"})
	}, nil)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, body)

	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, map[string]string{
		"path": "/testpage",
	}, resp.Cookies()...)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	errs, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Error 1", errs["field1"])
}

func TestInertia_SessionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSessionAdapter := inertiamocks.NewMockSessionAdapter[*inertiamocks.MockFiberSessionStore](ctrl)
	adapter := goinertia.NewFiberSessionAdapter[*inertiamocks.MockFiberSessionStore](mockSessionAdapter)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	mockSessionAdapter.EXPECT().Get(gomock.Any()).Return(nil, errors.New("some error")).Times(2)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.RedirectBackWithErrors(c, map[string]string{"field1": "Error 1"})
	}, nil)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, body)

	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, map[string]string{
		"path": "/testpage",
	}, resp.Cookies()...)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	errs, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, errs)
}

func TestInertia_RedirectBackWithValidationErrors(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.RedirectBackWithValidationErrors(c, goinertia.ValidationErrors{"field1": {"Error 1"}})
	}, nil)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, body)

	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, map[string]string{
		"path": "/testpage",
	}, resp.Cookies()...)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	errs, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Error 1", errs["field1"])
}

func TestInertia_InvalidContextKeyViewData(t *testing.T) {
	ta := inertiat.NewTestAppWithErrorHandler(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(c fiber.Ctx) error {
		c.Locals(goinertia.ContextKeyViewData, "invalid_check_error")

		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, body, "Something went wrong. Try again later")
}

func TestInertia_InvalidContextKeyViewData_CanDetails(t *testing.T) {
	ta := inertiat.NewTestAppWithErrorHandler(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoGet(func(c fiber.Ctx) error {
		c.Locals(goinertia.ContextKeyViewData, "invalid_check_error")

		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, body, "could not convert context view data to ma")
}

func TestInertia_RedirectBack(t *testing.T) {
	tests := []struct {
		name           string
		inertiaRequest bool
		referer        string
		page           string
		expectedURL    string
		expectedStatus int
	}{
		{
			name:           "inertia request with referer",
			inertiaRequest: true,
			referer:        "/previous-page",
			expectedURL:    "/previous-page",
			expectedStatus: fiber.StatusFound,
		},
		{
			name:           "inertia request without referer uses base URL",
			inertiaRequest: true,
			referer:        "",
			expectedURL:    "/test",
			expectedStatus: fiber.StatusFound,
		},
		{
			name:           "regular request with referer",
			inertiaRequest: false,
			referer:        "/previous-page",
			expectedURL:    "",
			expectedStatus: fiber.StatusFound,
		},
		{
			name:           "main page",
			inertiaRequest: false,
			page:           "/",
			referer:        "",
			expectedURL:    "",
			expectedStatus: fiber.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := inertiat.NewTestAppWithoutMiddleware(t)

			headers := map[string]string{}
			if tt.inertiaRequest {
				headers[goinertia.HeaderInertia] = "true"
			}

			if tt.referer != "" {
				headers["Referer"] = tt.referer
			}

			if tt.page != "" {
				headers["path"] = tt.page
			}

			//nolint:bodyclose // tests
			resp, _ := ta.DoGet(func(c fiber.Ctx) error {
				if tt.inertiaRequest {
					c.Set(goinertia.HeaderInertia, "true")
				}
				return ta.Inrt.RedirectBack(c)
			}, headers)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedURL != "" {
				assert.Equal(t, tt.expectedURL, resp.Header.Get(fiber.HeaderLocation))
			}
		})
	}
}

func TestInertia_WithComplexProps(t *testing.T) {
	ta := inertiat.NewTestAppWithErrorHandler(t)
	handler := func(c fiber.Ctx) error {
		ta.Inrt.WithFlash(c, goinertia.FlashLevelSuccess, "User created successfully")
		ta.Inrt.WithFlash(c, goinertia.FlashLevelInfo, "Please check your email")
		ta.Inrt.WithError(c, "fio", "Fio is required")
		ta.Inrt.WithErrors(c, map[string]string{
			"email":    "Email is required",
			"password": "Password must be at least 8 characters",
		})
		ta.Inrt.WithFlashOld(c, map[string]any{
			"email": "me@test.loc",
			"phone": "111123334455",
		})
		ta.Inrt.WithProp(c, "user", map[string]string{"name": "John"})
		ta.Inrt.WithProp(c, "title", "Test Page")
		ta.Inrt.WithProp(c, "count", 42)
		ta.Inrt.WithError(c, "phone", "Phone is required")
		ta.Inrt.WithViewData(c, "testViewDataKey", "test_view_data_VALUE_33")
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotContains(t, body, "test_view_data_VALUE_33")
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	flash, ok := page.Props["flash"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "User created successfully", flash["success"])
	assert.Equal(t, "Please check your email", flash["info"])
	errs, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Fio is required", errs["fio"])
	assert.Equal(t, "Phone is required", errs["phone"])
	assert.Equal(t, "Email is required", errs["email"])
	assert.Equal(t, "Password must be at least 8 characters", errs["password"])
	oldFlash, ok := page.Props["old"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "me@test.loc", oldFlash["email"])
	assert.Equal(t, "111123334455", oldFlash["phone"])

	//nolint:bodyclose // tests
	resp, body = ta.DoGet(handler, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, body, "test_view_data_VALUE_33")
	assert.Contains(t, body, "Fio is required")
	assert.Contains(t, body, "Phone is required")
	assert.Contains(t, body, "Email is required")
	assert.Contains(t, body, "Password must be at least 8 characters")
	assert.Contains(t, body, "User created successfully")
	assert.Contains(t, body, "Please check your email")
	assert.Contains(t, body, "me@test.loc")
	assert.Contains(t, body, "111123334455")
}

func TestInertia_WithComplexProps_WithRedirect(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))
	handler := func(c fiber.Ctx) error {
		ta.Inrt.WithFlash(c, goinertia.FlashLevelSuccess, "User created successfully")
		ta.Inrt.WithFlash(c, goinertia.FlashLevelInfo, "Please check your email")
		ta.Inrt.WithError(c, "fio", "Fio is required")
		ta.Inrt.WithErrors(c, map[string]string{
			"email":    "Email is required",
			"password": "Password must be at least 8 characters",
		})
		ta.Inrt.WithFlashOld(c, map[string]any{
			"email": "me@test.loc",
			"phone": "111123334455",
		})
		ta.Inrt.WithProp(c, "user", map[string]string{"name": "John"})
		ta.Inrt.WithProp(c, "title", "Test Page")
		ta.Inrt.WithProp(c, "count", 42)
		ta.Inrt.WithError(c, "phone", "Phone is required")
		ta.Inrt.WithViewData(c, "testViewDataKey", "test_view_data_VALUE_33")
		return ta.Inrt.RedirectBack(c)
	}

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaPost(handler, nil)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Empty(t, body)

	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil, resp.Cookies()...)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.NotContains(t, body, "test_view_data_VALUE_33")
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	flash, ok := page.Props["flash"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "User created successfully", flash["success"])
	assert.Equal(t, "Please check your email", flash["info"])
	errs, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Fio is required", errs["fio"])
	assert.Equal(t, "Phone is required", errs["phone"])
	assert.Equal(t, "Email is required", errs["email"])
	assert.Equal(t, "Password must be at least 8 characters", errs["password"])
	oldFlash, ok := page.Props["old"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "me@test.loc", oldFlash["email"])
	assert.Equal(t, "111123334455", oldFlash["phone"])

	//nolint:bodyclose // tests
	resp3, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}, nil)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.NotContains(t, body, "test_view_data_VALUE_33")
	page = inertiat.DecodePage(t, body)
	assert.Equal(t, "TestComponent", page.Component)
	assert.Equal(t, "John", page.Props["user"])
	_, ok = page.Props["flash"].(map[string]any)
	require.False(t, ok)
}

func TestInertia_FlashHelpers(t *testing.T) {
	tests := []struct {
		name     string
		method   func(*goinertia.Inertia, fiber.Ctx, string)
		key      string
		message  string
		expected string
	}{
		{
			name:     "WithFlashSuccess",
			method:   (*goinertia.Inertia).WithFlashSuccess,
			key:      "success",
			message:  "Operation successful",
			expected: "Operation successful",
		},
		{
			name:     "WithFlashInfo",
			method:   (*goinertia.Inertia).WithFlashInfo,
			key:      "info",
			message:  "Information message",
			expected: "Information message",
		},
		{
			name:     "WithFlashWarning",
			method:   (*goinertia.Inertia).WithFlashWarning,
			key:      "warning",
			message:  "Warning message",
			expected: "Warning message",
		},
		{
			name:     "WithFlashError",
			method:   (*goinertia.Inertia).WithFlashError,
			key:      "error",
			message:  "Error message",
			expected: "Error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := inertiat.NewTestAppWithoutMiddleware(t)
			//nolint:bodyclose // tests
			resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
				tt.method(ta.Inrt, c, tt.message)
				return ta.Inrt.Render(c, "TestComponent", map[string]any{})
			}, nil)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			page := inertiat.DecodePage(t, body)

			flash, ok := page.Props["flash"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.expected, flash[tt.key])
		})
	}
}

func TestInertia_Flash_WithXHR_Get(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	// First request: set flash and redirect back (simulate POST/redirect/GET pattern)
	//nolint:bodyclose // tests
	resp1, _ := ta.DoInertiaGet(func(c fiber.Ctx) error {
		ta.Inrt.WithFlashSuccess(c, "Saved!")
		return ta.Inrt.RedirectBack(c)
	}, map[string]string{
		"path": "/set",
	})
	require.Equal(t, http.StatusFound, resp1.StatusCode)

	// Second request: read page props (flash should be loaded from session)
	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Contains(t, resp2.Header.Get("Content-Type"), "application/json")
	assert.Contains(t, body, "\"flash\"")
	assert.Contains(t, body, "Saved!")
	assert.NotContains(t, body, "test_view_data_VALUE")

	// 3) Flash should be one-time; next request shouldn't include it again
	//nolint:bodyclose // tests
	resp3, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.NotContains(t, body, "Saved!")
}

func TestInertia_Flash_InvalidVersion_Get(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))
	method := http.MethodGet
	req := inertiat.NewInertiaRequest(method, "/test", nil)
	req.Header.Set(goinertia.HeaderVersion, "v0")
	//nolint:bodyclose // tests
	resp1, _ := ta.Do(method, "/test", func(c fiber.Ctx) error {
		ta.Inrt.WithFlashSuccess(c, "Saved!")
		return ta.Inrt.RedirectBack(c)
	}, req)
	require.Equal(t, http.StatusConflict, resp1.StatusCode)
	require.Equal(t, "http://localhost.loc:3000/test", resp1.Header.Get(goinertia.HeaderLocation))
}

func TestInertia_Flash_InvalidVersion_Post(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))
	method := http.MethodPost
	req := inertiat.NewInertiaRequest(method, "/test", nil)
	req.Header.Set(goinertia.HeaderVersion, "v0")
	//nolint:bodyclose // tests
	resp1, _ := ta.Do(method, "/test", func(c fiber.Ctx) error {
		ta.Inrt.WithFlashSuccess(c, "Saved!")
		return ta.Inrt.RedirectBack(c)
	}, req)
	require.Equal(t, http.StatusSeeOther, resp1.StatusCode)
	require.Equal(t, "/test", resp1.Header.Get(fiber.HeaderLocation))
}

func TestInertia_Flash_WithXHR_Post(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	// First request: set flash and redirect back (simulate POST/redirect/GET pattern)
	//nolint:bodyclose // tests
	resp1, _ := ta.DoInertiaPost(func(c fiber.Ctx) error {
		ta.Inrt.WithFlashSuccess(c, "Saved!")
		return ta.Inrt.RedirectBack(c)
	}, map[string]string{
		"path": "/set",
	})
	require.Equal(t, http.StatusSeeOther, resp1.StatusCode)

	// Second request: read page props (flash should be loaded from session)
	//nolint:bodyclose // tests
	resp2, body := ta.DoInertiaPost(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Contains(t, resp2.Header.Get("Content-Type"), "application/json")
	assert.Contains(t, body, "\"flash\"")
	assert.Contains(t, body, "Saved!")
	assert.NotContains(t, body, "test_view_data_VALUE")

	// 3) Flash should be one-time; next request shouldn't include it again
	//nolint:bodyclose // tests
	resp3, body := ta.DoInertiaPost(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.NotContains(t, body, "Saved!")
}

func TestInertia_Flash_WithoutXHR(t *testing.T) {
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(t, goinertia.WithSessionStore(adapter))

	// First request: set flash and redirect back (simulate POST/redirect/GET pattern)
	//nolint:bodyclose // tests
	resp1, _ := ta.DoPost(func(c fiber.Ctx) error {
		ta.Inrt.WithFlashSuccess(c, "Saved!")
		return ta.Inrt.RedirectBack(c)
	}, map[string]string{
		"path": "/set",
	})
	require.Equal(t, http.StatusFound, resp1.StatusCode)

	// Second request: read page props (flash should be loaded from session)
	//nolint:bodyclose // tests
	resp2, body := ta.DoGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Contains(t, resp2.Header.Get("Content-Type"), "text/html; charset=utf-8")
	assert.NotContains(t, body, "\"flash\"")
	assert.Contains(t, body, "Saved!")
	assert.Contains(t, body, "test_view_data_VALUE")

	// 3) Flash should be one-time; next request shouldn't include it again
	//nolint:bodyclose // tests
	resp3, body := ta.DoGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Page", map[string]any{"ok": true})
	}, map[string]string{
		"path": "/page",
	}, resp1.Cookies()...)
	require.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.NotContains(t, body, "Saved!")
}

func TestInertia_CSRFTokenProvider(t *testing.T) {
	t.Parallel()

	const tokenValue = "csrf-token-123" //nolint:gosec // tests
	const csrfNameToken = "csrf_token"
	ta := inertiat.NewTestApp(t, goinertia.WithCSRFTokenProvider(func(fiber.Ctx) (string, error) {
		return tokenValue, nil
	}), goinertia.WithCSRFPropName(csrfNameToken))

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}

	// Regular request should include CSRF token
	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, map[string]string{
		"path": "/csrf",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	csrfToken, ok := page.Props[csrfNameToken].(string)
	require.True(t, ok)
	assert.Equal(t, "csrf-token-123", csrfToken)

	// Partial reload should still include CSRF token
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		"path":                           "/csrf",
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page = inertiat.DecodePage(t, body)
	csrfToken, ok = page.Props[csrfNameToken].(string)
	require.True(t, ok)
	assert.Equal(t, "csrf-token-123", csrfToken)
}

func TestInertia_CSRFTokenProvider_Check(t *testing.T) {
	t.Parallel()

	const tokenValue = "csrf-token-123" //nolint:gosec // tests
	const csrfNameToken = "csrf_token"
	ta := inertiat.NewTestApp(t, goinertia.WithCSRFTokenProvider(func(c fiber.Ctx) (string, error) {
		c.Cookie(&fiber.Cookie{
			Name:     csrfNameToken,
			Value:    tokenValue,
			HTTPOnly: false,
			Domain:   "localhost.loc",
			SameSite: fiber.CookieSameSiteStrictMode,
			Secure:   true,
			MaxAge:   int((42 * time.Minute).Seconds()),
		})
		return "csrf-token-123", nil
	}), goinertia.WithCSRFPropName(csrfNameToken), goinertia.WithCSRFTokenCheckProvider(func(c fiber.Ctx) error {
		csrfTokenCookie := c.Cookies(csrfNameToken)
		assert.Equal(t, "csrf-token-123", csrfTokenCookie)
		return nil
	}))

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}

	handlerPost := func(c fiber.Ctx) error {
		return ta.Inrt.RedirectBack(c)
	}

	// Regular request should include CSRF token
	//nolint:bodyclose // tests
	resp1, body := ta.DoGet(handler, map[string]string{
		"path": "/csrf",
	})
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Contains(t, body, "csrf-token-123")
	//nolint:bodyclose // tests
	resp, body := ta.DoPost(handlerPost, map[string]string{
		"path": "/csrf",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, body)

	// Partial reload should still include CSRF token
	//nolint:bodyclose // tests
	resp, body = ta.DoPost(handlerPost, map[string]string{
		"path":                           "/csrf",
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Empty(t, body)
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handlerPost, map[string]string{
		"path":                           "/csrf",
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	csrfToken, ok := page.Props[csrfNameToken].(string)
	require.True(t, ok)
	assert.Equal(t, "csrf-token-123", csrfToken)
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaPost(handlerPost, map[string]string{
		"path":                           "/csrf",
		goinertia.HeaderPartialComponent: "TestComponent",
		goinertia.HeaderPartialOnly:      "user",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Empty(t, body)
}

func TestInertia_CSRFTokenProvider_CheckError(t *testing.T) {
	t.Parallel()

	const tokenValue = "csrf-token-123" //nolint:gosec // tests
	const csrfNameToken = "csrf_token"
	store := newTestStore()
	adapter := goinertia.NewFiberSessionAdapter[*session.Session](store)
	ta := inertiat.NewTestAppWithErrorHandler(
		t,
		goinertia.WithSessionStore(adapter),
		goinertia.WithCSRFTokenProvider(func(c fiber.Ctx) (string, error) {
			c.Cookie(&fiber.Cookie{
				Name:     csrfNameToken,
				Value:    tokenValue,
				HTTPOnly: false,
				Domain:   "localhost.loc",
				SameSite: fiber.CookieSameSiteStrictMode,
				Secure:   true,
				MaxAge:   int((42 * time.Minute).Seconds()),
			})
			return "csrf-token-123", nil
		}), goinertia.WithCSRFPropName(csrfNameToken), goinertia.WithCSRFTokenCheckProvider(func(fiber.Ctx) error {
			return goinertia.NewError(419, "CSRF invalid token")
		}),
	)

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	}

	handlerPost := func(c fiber.Ctx) error {
		return ta.Inrt.RedirectBack(c)
	}

	// Regular request should include CSRF token
	//nolint:bodyclose // tests
	resp1, body := ta.DoGet(handler, map[string]string{
		"path": "/csrf",
	})
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Contains(t, body, "csrf-token-123")
	//nolint:bodyclose // tests
	resp, body := ta.DoPost(handlerPost, map[string]string{
		"path": "/csrf",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Contains(t, body, "")
	//nolint:bodyclose // tests
	resp, body = ta.DoGet(handler, map[string]string{
		"path": "/csrf",
	}, resp.Cookies()...)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "The page expired, please try again")
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaPost(handlerPost, map[string]string{
		"path": "/csrf",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, body, "")
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handlerPost, map[string]string{
		"path": "/csrf",
	}, resp1.Cookies()...)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	csrfToken, ok := page.Props[csrfNameToken].(string)
	require.True(t, ok)
	assert.Equal(t, "csrf-token-123", csrfToken)
}

func TestInertia_CSRFTokenProvider_SkipWhenPropExists(t *testing.T) {
	t.Parallel()

	providerCalls := 0
	ta := inertiat.NewTestApp(t, goinertia.WithCSRFTokenProvider(func(fiber.Ctx) (string, error) {
		providerCalls++
		return "provider-token", nil
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			goinertia.ContextPropsCSRFToken: "preset-token",
		})
	}, map[string]string{
		"path": "/csrf-existing",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	csrfToken, ok := page.Props[goinertia.ContextPropsCSRFToken].(string)
	require.True(t, ok)
	assert.Equal(t, "preset-token", csrfToken)
	assert.Equal(t, 0, providerCalls)
}

func TestInertia_CSRFTokenProvider_Error(t *testing.T) {
	t.Parallel()

	ta := inertiat.NewTestApp(t, goinertia.WithCSRFTokenProvider(func(fiber.Ctx) (string, error) {
		return "", errors.New("no token")
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "Jane",
		})
	}, map[string]string{
		"path": "/csrf-error",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	_, ok := page.Props[goinertia.ContextPropsCSRFToken]
	assert.False(t, ok)
}

func TestInertia_CSRFTokenProvider_Empty(t *testing.T) {
	t.Parallel()

	ta := inertiat.NewTestApp(t, goinertia.WithCSRFTokenProvider(func(fiber.Ctx) (string, error) {
		return "", nil
	}))

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "TestComponent", map[string]any{
			"user": "Jane",
		})
	}, map[string]string{
		"path": "/csrf-error",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	_, ok := page.Props[goinertia.ContextPropsCSRFToken]
	assert.True(t, ok)
}

func TestInertia_PartialExcept_Precedence(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", map[string]any{
			"foo": "a",
			"bar": "b",
			"baz": "c",
		})
	}, map[string]string{
		goinertia.HeaderPartialComponent: "Home",
		goinertia.HeaderPartialOnly:      "bar",
		goinertia.HeaderPartialExcept:    "foo",
	})

	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "b", page.Props["bar"])
	assert.Equal(t, "c", page.Props["baz"])
	_, ok := page.Props["foo"]
	assert.False(t, ok)
}

func TestInertia_DeferredProps(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)
	var calls int32

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", map[string]any{
			"eager": "ok",
			"lazy": goinertia.Defer(goinertia.LazyProp{
				Key: "lazy",
				Fn: func(_ context.Context) (any, error) {
					atomic.AddInt32(&calls, 1)
					return "value", nil
				},
			}),
		})
	}

	// Full response should not include deferred prop.
	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	_, ok := page.Props["lazy"]
	assert.False(t, ok)
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls))
	group, ok := page.DeferredProps["default"]
	require.True(t, ok)
	assert.Contains(t, group, "lazy")

	// Explicit partial reload should include deferred prop.
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		goinertia.HeaderPartialComponent: "Home",
		goinertia.HeaderPartialOnly:      "lazy",
	})
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page = inertiat.DecodePage(t, body)
	assert.Equal(t, "value", page.Props["lazy"])
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestInertia_OnceProps(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", map[string]any{
			"plans": goinertia.Once("basic", goinertia.WithOnceKey("plans_v1")),
		})
	}

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Equal(t, "basic", page.Props["plans"])
	cfg, ok := page.OnceProps["plans_v1"]
	require.True(t, ok)
	assert.Equal(t, "plans", cfg.Prop)

	// Skip once prop if client already has it.
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		goinertia.HeaderExceptOnceProps: "plans_v1",
	})
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page = inertiat.DecodePage(t, body)
	_, ok = page.Props["plans"]
	assert.False(t, ok)
	_, ok = page.OnceProps["plans_v1"]
	assert.True(t, ok)
}

func TestInertia_ErrorBag(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(func(c fiber.Ctx) error {
		ta.Inrt.WithErrors(c, map[string]string{"email": "Invalid"})
		return ta.Inrt.Render(c, "Home", map[string]any{})
	}, map[string]string{
		goinertia.HeaderErrorBag: "login",
	})

	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	errorsProp, ok := page.Props["errors"].(map[string]any)
	require.True(t, ok)
	bag, ok := errorsProp["login"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Invalid", bag["email"])
}

func TestInertia_ResetMergeProps(t *testing.T) {
	ta := inertiat.NewTestAppWithoutMiddleware(t)

	handler := func(c fiber.Ctx) error {
		return ta.Inrt.Render(c, "Home", map[string]any{
			"items": goinertia.Merge([]int{1, 2, 3}),
		})
	}

	//nolint:bodyclose // tests
	resp, body := ta.DoInertiaGet(handler, nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page := inertiat.DecodePage(t, body)
	assert.Contains(t, page.MergeProps, "items")

	// Reset should remove merge metadata for this response.
	//nolint:bodyclose // tests
	resp, body = ta.DoInertiaGet(handler, map[string]string{
		goinertia.HeaderReset: "items",
	})
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	page = inertiat.DecodePage(t, body)
	assert.NotContains(t, page.MergeProps, "items")
}

func newTestStore() *session.Store {
	_, store := session.NewWithStore(session.Config{
		IdleTimeout:       10 * time.Minute,
		AbsoluteTimeout:   15 * time.Minute,
		CookiePath:        "/",
		CookieDomain:      "localhost.loc",
		CookieSessionOnly: false,
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "Lax",
		Extractor:         extractors.FromCookie("sess_id"),
		// KeyLookup:         "cookie:sess_id",
	})
	return store
}
