package inertiat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
)

type TestApp struct {
	Test   *testing.T
	App    *fiber.App
	Inrt   *goinertia.Inertia
	IsProd bool
}

func NewTestAppWithoutMiddleware(t *testing.T, opts ...goinertia.Option) TestApp {
	t.Helper()

	inrt := NewForTest("http://localhost.loc:3000", opts...)
	app := fiber.New()

	return TestApp{Test: t, App: app, Inrt: inrt}
}

func NewTestApp(t *testing.T, opts ...goinertia.Option) TestApp {
	t.Helper()

	ta := NewTestAppWithoutMiddleware(t, opts...)
	ta.App.Use(ta.Inrt.Middleware())

	return ta
}

func NewTestAppWithErrorHandler(t *testing.T, opts ...goinertia.Option) TestApp {
	t.Helper()

	ta := NewTestAppWithoutMiddleware(t, opts...)
	ta.App = fiber.New(fiber.Config{ErrorHandler: ta.Inrt.MiddlewareErrorListener()})
	ta.App.Use(ta.Inrt.Middleware())

	return ta
}

func getPath(hs ...map[string]string) string {
	for _, h := range hs {
		if val, ok := h["path"]; ok {
			delete(h, "path")
			return val
		}
	}

	return "/test"
}

func (ta TestApp) DoGet(
	handler func(c fiber.Ctx) error,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodGet
	path := getPath(headers)
	req := NewRequest(method, path, nil, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) DoPost(
	handler func(c fiber.Ctx) error,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodPost
	path := getPath(headers)
	req := NewRequest(method, path, nil, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) DoInertiaGet(
	handler func(c fiber.Ctx) error,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodGet
	path := getPath(headers)
	req := NewInertiaRequest(method, path, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) DoInertiaPost(
	handler func(c fiber.Ctx) error,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodPost
	path := getPath(headers)
	req := NewInertiaRequest(method, path, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) DoPostBody(
	handler func(c fiber.Ctx) error,
	body io.Reader,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodPost
	path := getPath(headers)
	req := NewRequest(method, path, body, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) DoInertiaPostBody(
	handler func(c fiber.Ctx) error,
	body io.Reader,
	headers map[string]string,
	cookies ...*http.Cookie,
) (*http.Response, string) {
	method := http.MethodPost
	path := getPath(headers)
	req := NewInertiaRequestBody(method, path, body, headers, cookies...)
	return ta.Do(method, path, handler, req)
}

func (ta TestApp) Do(
	method string,
	path string,
	handler func(c fiber.Ctx) error,
	req *http.Request,
) (*http.Response, string) {
	ta.App.Add([]string{method}, path, handler)
	resp := Do(ta.Test, ta.App, req)
	body := ReadBody(ta.Test, resp)
	// Mark body as consumed to make intent explicit for callers and tools.
	resp.Body = http.NoBody

	return resp, body
}

// NewInertiaRequest creates a request with X-Inertia set to true. Additional headers can override defaults.
func NewInertiaRequest(
	method string,
	target string,
	headers map[string]string,
	cookies ...*http.Cookie,
) *http.Request {
	if headers == nil {
		headers = map[string]string{}
	}
	headers[goinertia.HeaderInertia] = "true"
	headers[goinertia.HeaderVersion] = "v1.0"
	return NewRequest(method, target, nil, headers, cookies...)
}

// NewInertiaRequestBody creates an Inertia request with body.
func NewInertiaRequestBody(
	method string,
	target string,
	body io.Reader,
	headers map[string]string,
	cookies ...*http.Cookie,
) *http.Request {
	if headers == nil {
		headers = map[string]string{}
	}
	headers[goinertia.HeaderInertia] = "true"
	return NewRequest(method, target, body, headers, cookies...)
}

// NewRequest constructs a basic HTTP request and applies headers.
func NewRequest(
	method string,
	target string,
	body io.Reader,
	headers map[string]string,
	cookies ...*http.Cookie,
) *http.Request {
	req := httptest.NewRequest(method, target, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// Do executes request against app and fails the test on error.
func Do(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	return resp
}

// ReadBody reads the full response body as string and closes it.
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	return string(b)
}

// DecodePage decodes a JSON X-Inertia response into PageDTO and closes body.
func DecodePage(t *testing.T, body string) goinertia.PageDTO {
	t.Helper()

	var page goinertia.PageDTO
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("failed to decode inertia page: %v", err)
	}

	return page
}

// CreateBody creating body.
func CreateBody(
	t *testing.T,
	fieldName string,
	fileName string,
	fileContents ...string,
) (*bytes.Buffer, *multipart.Writer) {
	t.Helper()

	fnFileName := func(number int) string {
		if fileName != "" {
			return fmt.Sprintf(fileName, number)
		}

		return fmt.Sprintf("file-%d.txt", number)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for i, fc := range fileContents {
		part, err := writer.CreateFormFile(fieldName, fnFileName(i))
		require.NoError(t, err)
		_, err = io.WriteString(part, fc)
		require.NoError(t, err)
	}
	err := writer.Close()
	require.NoError(t, err)

	return body, writer
}

// ReadFileContents read files.
func ReadFileContents(t *testing.T, filePath string) [][]byte {
	t.Helper()
	files, err := filepath.Glob(filePath)
	require.NoError(t, err)

	uploadedContents := make([][]byte, 0, len(files))
	for _, file := range files {
		uploadedContent, err := os.ReadFile(file)
		require.NoError(t, err)
		uploadedContents = append(uploadedContents, uploadedContent)
	}

	return uploadedContents
}

// ReadFileNames read file names.
func ReadFileNames(t *testing.T, filePath string) []string {
	t.Helper()
	files, err := filepath.Glob(filePath)
	require.NoError(t, err)

	names := make([]string, 0, len(files))
	for _, file := range files {
		_, err := os.Stat(file)
		require.NoError(t, err)
		names = append(names, file)
	}

	return names
}
