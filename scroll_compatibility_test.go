package goinertia_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
)

func TestScrollPositionalConsumer(t *testing.T) {
	t.Parallel()
	//nolint:gosec // Fixed local compiler fixture and temporary output; no external input.
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(t.TempDir(), "consumer"),
		"testdata/positional-consumer/main.go")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func TestScrollResetWireCompatibility(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, reset := range []string{"", "posts", "other"} {
				raw := renderReplayPage(t, adapter, map[string]string{core.HeaderPartialOnly: "posts", core.HeaderReset: reset}, 2)
				var page struct {
					ScrollProps map[string]struct {
						core.ScrollPropConfig
						Reset bool `json:"reset"`
					} `json:"scrollProps"`
				}
				require.NoError(t, json.Unmarshal(raw, &page))
				cfg, ok := page.ScrollProps["posts"]
				require.True(t, ok)
				require.Equal(t, "page", cfg.PageName)
				require.Equal(t, reset == "posts", cfg.Reset)
			}
		})
	}
}
