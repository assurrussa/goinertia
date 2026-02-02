package inertiat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/inertiat"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		assert.NotNil(t, inertiat.NewForTest(
			"http://localhost:3000",
			goinertia.WithRootHotTemplate("public/static/hot"),
		))
	})
}
