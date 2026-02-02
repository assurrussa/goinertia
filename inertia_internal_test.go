package goinertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/inertiat/fibert"
	"github.com/assurrussa/goinertia/testdata"
	"github.com/assurrussa/goinertia/views"
)

func TestInertia_TemplateError_RootTmplFSError(t *testing.T) {
	inrt := New("http://localhost:3000")
	inrt.templateFS = nil
	inrt.rootTemplate = ""
	inrt.rootErrorTemplate = ""

	tmpl, err := inrt.createRootTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)

	tmpl, err = inrt.createRootErrorTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)

	inrt.templateFS = views.Templates
	tmpl, err = inrt.createRootTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)

	tmpl, err = inrt.createRootErrorTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)
}

func TestInertia_Template_RenderHTMLError(t *testing.T) {
	inrt := New(
		"http://localhost:3000",
		WithFS(testdata.Files),
		WithPublicFS(testdata.Files),
	)

	appCtx := fibert.Default()
	errExpectCause := errors.New("test2")
	appErr := NewError(400, "error message", errExpectCause)

	err := inrt.renderHTMLError(appCtx, appErr, errExpectCause.Error())
	require.NoError(t, err)
	assert.Equal(t, appErr.Code, appCtx.Response().StatusCode())
	body := appCtx.Response().Body()
	assert.Contains(t, string(body), appErr.Message)
	assert.Contains(t, string(body), errExpectCause.Error())
}

func TestInertia_Template_RenderHTMLError_AppErrNil(t *testing.T) {
	inrt := New(
		"http://localhost:3000",
		WithFS(testdata.Files),
		WithPublicFS(testdata.Files),
	)

	appCtx := fibert.Default()

	err := inrt.renderHTMLError(appCtx, nil, "")
	require.NoError(t, err)
	assert.Equal(t, 500, appCtx.Response().StatusCode())
	body := appCtx.Response().Body()
	assert.Contains(t, string(body), `">500</`)
	assert.Contains(t, string(body), `inertia: value is nil`)
}

func TestInertia_TemplateError(t *testing.T) {
	inrt := New("http://localhost:3000")
	inrt.templateFS = nil
	inrt.rootTemplate = ""

	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		return inrt.Render(c, "TestComponent", map[string]any{
			"user": "John",
		})
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
