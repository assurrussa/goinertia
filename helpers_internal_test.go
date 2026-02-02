package goinertia

import (
	"html/template"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Marshal(t *testing.T) {
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
	path, err := asset("test/test2/any.txt")
	require.NoError(t, err)
	tassert.Equal(t, `/public/dist/test/test2/any.txt`, path)
}

func Test_Raw(t *testing.T) {
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
