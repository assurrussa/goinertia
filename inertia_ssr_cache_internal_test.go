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

	cache.mu.Lock()
	defer cache.mu.Unlock()
	assert.LessOrEqual(t, len(cache.items), 2)
}

func TestSSRCache_LRU(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(100*time.Millisecond, 2)
	require.NotNil(t, cache)

	cache.Set("key-1", &SsrDTO{Body: "body-1"})
	cache.Set("key-2", &SsrDTO{Body: "body-2"})

	_, ok := cache.Get("key-1") // key-2 becomes LRU
	require.True(t, ok)

	cache.Set("key-3", &SsrDTO{Body: "body-3"})

	_, ok = cache.Get("key-2")
	assert.False(t, ok)
	_, ok = cache.Get("key-1")
	assert.True(t, ok)
	_, ok = cache.Get("key-3")
	assert.True(t, ok)
}

func TestSSRCache_EvictExpiredBeforeLRU(t *testing.T) {
	t.Parallel()

	cache := newSSRCache(40*time.Millisecond, 2)
	require.NotNil(t, cache)

	cache.Set("key-1", &SsrDTO{Body: "body-1"})
	time.Sleep(25 * time.Millisecond)

	cache.Set("key-2", &SsrDTO{Body: "body-2"})
	time.Sleep(25 * time.Millisecond) // key-1 expired, key-2 still valid

	cache.Set("key-3", &SsrDTO{Body: "body-3"})

	_, ok := cache.Get("key-1")
	assert.False(t, ok)
	_, ok = cache.Get("key-2")
	assert.True(t, ok)
	_, ok = cache.Get("key-3")
	assert.True(t, ok)
}
