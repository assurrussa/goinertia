package goinertia

import (
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/inertiat/fibert"
	"github.com/assurrussa/goinertia/testdata"
	"github.com/assurrussa/goinertia/views"
)

func TestInertia_CreateAndValidation(t *testing.T) {
	t.Parallel()

	opts := []Option{
		WithFS(testdata.Files),
		WithPublicFS(testdata.Files),
		WithSetSharedFuncMap(template.FuncMap{"embed": func(val string) string {
			return "embed-test123" + val + ".html"
		}}),
	}

	inrt, err := NewWithValidation("")
	require.ErrorIs(t, err, ErrBaseURLEmpty)
	require.Nil(t, inrt)

	inrt, err = NewWithValidation("http://localhost:3000")
	require.Error(t, err)
	require.Nil(t, inrt)

	inrt, err = NewWithValidation("http://localhost:3000", opts...)
	require.NoError(t, err)
	require.NotEmpty(t, inrt)

	assert.Panics(t, func() {
		Must(NewWithValidation("http://localhost:3000"))
	})

	assert.NotPanics(t, func() {
		Must(NewWithValidation("http://localhost:3000", opts...))
	})
}

func TestInertia_TemplateError_RootTmplFSError(t *testing.T) {
	t.Parallel()

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

	// Verify sticky error: calling again should return the same error
	tmpl, err = inrt.createRootTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)

	// To test successful initialization, we must use a new instance because sync.Once is already done.
	inrt2 := New("http://localhost:3000")
	inrt2.templateFS = views.Templates
	inrt2.rootTemplate = ""
	tmpl, err = inrt2.createRootTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)

	inrt2.rootErrorTemplate = ""
	tmpl, err = inrt2.createRootErrorTemplate()
	require.Error(t, err)
	assert.Nil(t, tmpl)
}

func TestInertia_Template_RenderHTMLError(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestInertia_DevMode_HotFile(t *testing.T) {
	tmpDir := t.TempDir()
	hotFile := filepath.Join(tmpDir, "hot")
	err := os.WriteFile(hotFile, []byte("http://localhost:3000"), 0o600)
	require.NoError(t, err)
	publicFS, ok := os.DirFS(tmpDir).(fs.ReadFileFS)
	require.True(t, ok)
	updateHotFile := func(content string) {
		err := os.WriteFile(hotFile, []byte(content), 0o600)
		require.NoError(t, err)
	}

	// Case 1: Without Dev Mode (Cached)
	t.Run("CachedWithoutDevMode", func(t *testing.T) {
		// Reset file
		updateHotFile("http://localhost:3000")

		inrt := New("http://example.com", WithPublicFS(publicFS), WithRootHotTemplate("hot"))

		// First read
		assert.Equal(t, "http://localhost:3000", inrt.hotServerURL())

		// Update file
		updateHotFile("http://localhost:5000")

		// Should still be old value because of caching
		assert.Equal(t, "http://localhost:3000", inrt.hotServerURL())
	})

	// Case 2: With Dev Mode (Not Cached)
	t.Run("NotCachedWithDevMode", func(t *testing.T) {
		// Reset file
		updateHotFile("http://localhost:3000")

		inrt := New("http://example.com", WithPublicFS(publicFS), WithRootHotTemplate("hot"), WithDevMode())

		// First read
		assert.Equal(t, "http://localhost:3000", inrt.hotServerURL())

		// Update file
		updateHotFile("http://localhost:5000")

		// Should be new value
		assert.Equal(t, "http://localhost:5000", inrt.hotServerURL())
	})
}

func TestInertia_DevMode_Templates(t *testing.T) {
	// Setup temp dir and file
	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "app.gohtml")

	// Initial content
	err := os.WriteFile(tmplPath, []byte("<html>{{.page.Component}}</html>"), 0o600)
	require.NoError(t, err)

	// Helper
	updateTmpl := func(content string) {
		err := os.WriteFile(tmplPath, []byte(content), 0o600)
		require.NoError(t, err)
	}

	viewFS := os.DirFS(tmpDir)

	// Case 1: Without Dev Mode (Cached)
	t.Run("CachedWithoutDevMode", func(t *testing.T) {
		updateTmpl("<html>Old</html>")
		inrt := New("http://example.com", WithFS(viewFS), WithRootTemplate("app.gohtml"))

		// Mock context
		c := fibert.Default()

		// First render
		err := inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.Contains(t, string(c.Response().Body()), "<html>Old</html>")

		// Update template
		updateTmpl("<html>New</html>")

		// Second render - should be Old
		c = fibert.Default()
		err = inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.Contains(t, string(c.Response().Body()), "<html>Old</html>")
	})

	// Case 2: With Dev Mode (Not Cached)
	t.Run("NotCachedWithDevMode", func(t *testing.T) {
		updateTmpl("<html>Old</html>")
		inrt := New("http://example.com", WithFS(viewFS), WithRootTemplate("app.gohtml"), WithDevMode())

		// Mock context
		c := fibert.Default()

		// First render
		err := inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.Contains(t, string(c.Response().Body()), "<html>Old</html>")

		// Update template
		updateTmpl("<html>New</html>")

		// Second render - should be New
		c = fibert.Default()
		err = inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.Contains(t, string(c.Response().Body()), "<html>New</html>")
	})
}
