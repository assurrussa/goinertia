package goinertia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSRCache_Nil(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(0, 0)
	require.Nil(t, cache)

	cache = newSSRCache(-1, -1)
	require.Nil(t, cache)

	var c *ssrCache

	val, ok := c.Get("key-1")
	require.False(t, ok)
	require.Nil(t, val)

	assert.NotPanics(t, func() {
		c.Set("key-1", &SsrDTO{Body: "body-1"})
	})
}

func TestSSRCache_Clone(t *testing.T) {
	t.Parallel()

	c := cloneSSR(nil)
	require.Nil(t, c)
}

func TestSSRCache_GetSet(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(100*time.Millisecond, 2)
	require.NotNil(t, cache)

	value := &SsrDTO{
		Head: []string{"head-1"},
		Body: "body-1",
	}
	cache.Set("key-1", value)

	got, ok := cache.Get("key-1")
	require.True(t, ok)
	require.NotNil(t, got)
	got2, ok2 := cache.Get("key-2")
	require.False(t, ok2)
	require.Nil(t, got2)
	assert.Equal(t, "body-1", got.Body)
	assert.Equal(t, []string{"head-1"}, got.Head)

	got.Head[0] = "mutated"
	gotAgain, ok := cache.Get("key-1")
	require.True(t, ok)
	assert.Equal(t, []string{"head-1"}, gotAgain.Head)
}

func TestSSRCache_Expire(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(20*time.Millisecond, 2)
	require.NotNil(t, cache)

	cache.Set("key-1", &SsrDTO{Body: "body-1"})
	time.Sleep(30 * time.Millisecond)

	_, ok := cache.Get("key-1")
	assert.False(t, ok)

	cache.Set("key-1", &SsrDTO{Body: "body-1"})
	cache.Set("key-2", &SsrDTO{Body: "body-2"})
	time.Sleep(30 * time.Millisecond)
	cache.Set("key-1", &SsrDTO{Body: "body-1"})
}

func TestSSRCache_MaxEntries(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(100*time.Millisecond, 2)
	require.NotNil(t, cache)

	cache.Set("key-1", &SsrDTO{Body: "body-1"})
	cache.Set("key-2", &SsrDTO{Body: "body-2"})
	cache.Set("key-3", &SsrDTO{Body: "body-3"})

	cache.mu.RLock()
	defer cache.mu.RUnlock()
	assert.LessOrEqual(t, len(cache.items), 2)
}
