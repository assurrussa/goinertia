package goinertia

import (
	"context"
	"html/template"
	"strconv"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/inertiat/fibert"
)

func Test_Marshal(t *testing.T) {
	t.Parallel()

	obj := struct {
		Foo string `json:"foo"`
	}{
		Foo: "bar",
	}
	js, err := marshal(&obj)
	require.NoError(t, err)
	tassert.JSONEq(t, `{"foo":"bar"}`, string(js))
}

func Test_Asset(t *testing.T) {
	t.Parallel()

	path, err := asset("test/test2/any.txt")
	require.NoError(t, err)
	tassert.Equal(t, `/public/dist/test/test2/any.txt`, path)
}

func Test_Raw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value   any
		want    string
		wantErr bool
	}{
		{value: "any-text", want: "any-text"},
		{value: []string{"any-text", "any-text2"}, want: "any-text\nany-text2"},
		{value: []string{}, want: ""},
		{value: nil, want: "", wantErr: true},
		{value: true, want: "", wantErr: true},
		{value: 123, want: "", wantErr: true},
	}

	for i, tt := range tests {
		tt := tt
		t.Run("case #"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			path, err := raw(tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			//nolint:gosec // G203: The used method does not auto-escape HTML - tests
			tassert.Equal(t, template.HTML(tt.want), path)
		})
	}
}

func Test_Sleeper(t *testing.T) {
	t.Parallel()

	require.NoError(t, sleeper(t.Context(), 10*time.Millisecond))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, sleeper(ctx, 10*time.Millisecond), context.Canceled)
}

func Test_AppendUnique(t *testing.T) {
	t.Parallel()

	list := []string{"a", "b"}
	result := appendUnique(list, "b")
	require.Len(t, result, 2)
	tassert.Equal(t, []string{"a", "b"}, result)

	result = appendUnique(list, "c")
	require.Len(t, result, 3)
	tassert.Equal(t, []string{"a", "b", "c"}, result)
}

func Test_AddVaryHeader(t *testing.T) {
	t.Parallel()

	t.Run("empty value", func(t *testing.T) {
		t.Parallel()

		ctx := fibert.Default()
		addVaryHeader(ctx, "")
		tassert.Empty(t, string(ctx.Response().Header.Peek("Vary")))
	})

	t.Run("initial set and dedupe", func(t *testing.T) {
		t.Parallel()

		ctx := fibert.Default()
		addVaryHeader(ctx, "X-Inertia")
		tassert.Equal(t, "X-Inertia", string(ctx.Response().Header.Peek("Vary")))

		addVaryHeader(ctx, "X-Inertia")
		tassert.Equal(t, "X-Inertia", string(ctx.Response().Header.Peek("Vary")))
	})

	t.Run("append new value", func(t *testing.T) {
		t.Parallel()

		ctx := fibert.Default()
		addVaryHeader(ctx, "X-Inertia")
		addVaryHeader(ctx, "Precognition")
		tassert.Equal(t, "X-Inertia, Precognition", string(ctx.Response().Header.Peek("Vary")))
	})

	t.Run("case-insensitive dedupe", func(t *testing.T) {
		t.Parallel()

		ctx := fibert.Default()
		ctx.Set("Vary", "x-inertia")
		addVaryHeader(ctx, "X-Inertia")
		tassert.Equal(t, "x-inertia", string(ctx.Response().Header.Peek("Vary")))
	})
}

func Test_buildInertiaLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseURL     string
		originalURL string
		want        string
	}{
		{
			name:        "absolute base, trailing slash, absolute path + query",
			baseURL:     "http://localhost.loc:3000/",
			originalURL: "/test?x=1",
			want:        "http://localhost.loc:3000/test?x=1",
		},
		{
			name:        "absolute base, no trailing slash, absolute path + query",
			baseURL:     "http://localhost.loc:3000",
			originalURL: "/test?x=1",
			want:        "http://localhost.loc:3000/test?x=1",
		},
		{
			name:        "absolute base, trailing slash, relative path + query",
			baseURL:     "http://localhost.loc:3000/",
			originalURL: "test?x=1",
			want:        "http://localhost.loc:3000/test?x=1",
		},
		{
			name:        "absolute base, base path preserved for relative reference",
			baseURL:     "http://localhost.loc:3000/app/",
			originalURL: "test?x=1",
			want:        "http://localhost.loc:3000/app/test?x=1",
		},
		{
			name:        "absolute base, empty original defaults to /",
			baseURL:     "http://localhost.loc:3000/",
			originalURL: "",
			want:        "http://localhost.loc:3000/",
		},
		{
			name:        "non-absolute base falls back to concat",
			baseURL:     "localhost.loc:3000/",
			originalURL: "/test?x=1",
			want:        "/test?x=1",
		},
		{
			name:        "empty original url",
			baseURL:     "http://localhost.loc:3000",
			originalURL: "",
			want:        "http://localhost.loc:3000",
		},
		{
			name:        "invalid original url falls back to concat",
			baseURL:     "http://localhost.loc:3000/",
			originalURL: "/test\n",
			want:        "http://localhost.loc:3000/",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := parseInertiaBaseURL(tt.baseURL)
			require.Equal(t, tt.want, buildInertiaLocation(base, tt.originalURL))
		})
	}
}

var buildInertiaLocationSink string

const buildInertiaLocationBenchOriginalURL = "/test?x=1&y=2"

func Benchmark_buildInertiaLocation_Absolute(b *testing.B) {
	baseURL := "http://localhost.loc:3000/"
	originalURL := buildInertiaLocationBenchOriginalURL
	base := parseInertiaBaseURL(baseURL)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buildInertiaLocationSink = buildInertiaLocation(base, originalURL)
	}
}

func Benchmark_buildInertiaLocation_RelativeRef(b *testing.B) {
	baseURL := "http://localhost.loc:3000/app/"
	originalURL := "test?x=1&y=2"
	base := parseInertiaBaseURL(baseURL)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buildInertiaLocationSink = buildInertiaLocation(base, originalURL)
	}
}

func Benchmark_buildInertiaLocation_NonAbsoluteFallback(b *testing.B) {
	baseURL := "localhost.loc:3000/"
	originalURL := buildInertiaLocationBenchOriginalURL
	base := parseInertiaBaseURL(baseURL)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buildInertiaLocationSink = buildInertiaLocation(base, originalURL)
	}
}

func Benchmark_buildInertiaLocation_NaiveConcat(b *testing.B) {
	baseURL := "http://localhost.loc:3000/"
	originalURL := buildInertiaLocationBenchOriginalURL

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buildInertiaLocationSink = baseURL + originalURL
	}
}

func Test_NormalizeValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		tassert.Nil(t, normalizeValidationErrors(nil))
	})

	t.Run("validation errors copy", func(t *testing.T) {
		t.Parallel()

		orig := ValidationErrors{"email": {"Invalid"}}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid"}, res["email"])

		orig["email"][0] = "Changed"
		tassert.Equal(t, []string{"Invalid"}, res["email"])
	})

	t.Run("map string slice", func(t *testing.T) {
		t.Parallel()

		orig := map[string][]string{"email": {"Invalid"}}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid"}, res["email"])

		orig["email"][0] = "Changed"
		tassert.Equal(t, []string{"Invalid"}, res["email"])
	})

	t.Run("map string string", func(t *testing.T) {
		t.Parallel()

		orig := map[string]string{"email": "Invalid"}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid"}, res["email"])
	})

	t.Run("map string map string string", func(t *testing.T) {
		t.Parallel()

		orig := map[string]map[string]string{
			"bag1": {"email": "Invalid"},
			"bag2": {"name": "Required"},
		}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid"}, res["email"])
		tassert.Equal(t, []string{"Required"}, res["name"])
	})

	t.Run("empty map string map string string", func(t *testing.T) {
		t.Parallel()

		orig := map[string]map[string]string{}
		res := normalizeValidationErrors(orig)
		tassert.Nil(t, res)
	})

	t.Run("map string any", func(t *testing.T) {
		t.Parallel()

		orig := map[string]any{
			"email": "Invalid",
			"bag": map[string]string{
				"name": "Required",
			},
		}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid"}, res["email"])
		tassert.Equal(t, []string{"Required"}, res["name"])
	})

	t.Run("map string any (json-unmarshal style arrays)", func(t *testing.T) {
		t.Parallel()

		orig := map[string]any{
			"email": []any{"Invalid", "Too short"},
			"bag": map[string]any{
				"name": []any{"Required"},
			},
		}
		res := normalizeValidationErrors(orig)
		require.NotNil(t, res)
		tassert.Equal(t, []string{"Invalid", "Too short"}, res["email"])
		tassert.Equal(t, []string{"Required"}, res["name"])
	})
}

func Test_IsPrecognition(t *testing.T) {
	ctx := fibert.Default()
	tassert.False(t, IsPrecognition(ctx))

	ctx.Request().Header.Set(HeaderPrecognition, "true")
	tassert.True(t, IsPrecognition(ctx))
}

func Test_FlattenValidationErrors(t *testing.T) {
	t.Parallel()

	input := ValidationErrors{
		"email": {"Invalid"},
		"name":  {},
	}
	flat := flattenValidationErrors(input)
	require.NotNil(t, flat)
	tassert.Equal(t, "Invalid", flat["email"])
	_, ok := flat["name"]
	tassert.False(t, ok)
}
