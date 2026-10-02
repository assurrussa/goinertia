package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultSSRURL          = "http://127.0.0.1:13714/render"
	DefaultSSRTimeout      = 3 * time.Second
	DefaultCacheTTL        = 5 * time.Minute
	DefaultCacheMaxEntries = 1024
	DefaultSSRMaxRetries   = 1
	DefaultSSRRetryDelay   = 10 * time.Millisecond
)

type SSRConfig struct {
	URL             string
	Timeout         time.Duration
	Headers         map[string]string
	CacheTTL        time.Duration
	CacheMaxEntries int
	SSRClient       SSRClient
	MaxRetries      int
	RetryDelay      time.Duration
	RetryStatuses   []int
	DisableRetries  bool
}

type defaultSSRClient struct {
	client *http.Client
}

func (c *defaultSSRClient) Reset() {
	c.client.CloseIdleConnections()
}

func (c *defaultSSRClient) Post(ctx context.Context, url string, body []byte, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	owned, err := io.ReadAll(resp.Body)
	return resp.StatusCode, owned, err
}

func (i *Inertia) IsSSREnabled() bool {
	return i.ssrConfig.URL != "" && i.ssrClient != nil
}

func (i *Inertia) EnableSSR(cfg SSRConfig) {
	i.ssrConfig = normalizeSSRConfig(cfg)
	if i.ssrConfig.URL == "" {
		i.DisableSSR()
		return
	}

	if i.ssrConfig.SSRClient != nil {
		i.ssrClient = i.ssrConfig.SSRClient
	} else if i.ssrClient == nil {
		i.ssrClient = &defaultSSRClient{client: &http.Client{}}
	}

	i.initSSRCache()
}

func (i *Inertia) EnableSSRWithDefault() {
	i.EnableSSR(SSRConfig{
		URL:             DefaultSSRURL,
		Timeout:         DefaultSSRTimeout,
		CacheTTL:        DefaultCacheTTL,
		CacheMaxEntries: DefaultCacheMaxEntries,
	})
}

func (i *Inertia) DisableSSR() {
	i.ssrConfig = SSRConfig{}
	if i.ssrClient != nil {
		i.ssrClient.Reset()
		i.ssrClient = nil
	}
	i.ssrCache = nil
}

func (i *Inertia) ProcessSSR(ctx context.Context, page *PageDTO) (*SsrDTO, error) {
	return i.processSSR(ctx, page)
}

func (i *Inertia) processSSR(ctx context.Context, page any) (*SsrDTO, error) {
	if !i.IsSSREnabled() {
		return nil, nil //nolint:nilnil // is need
	}

	var err error

	js, err := json.Marshal(page)
	if err != nil {
		i.logger.ErrorContext(ctx, "SSR marshal failed", "error", err)
		return nil, fmt.Errorf("error marshaling page: %w", err)
	}

	cacheKey := ssrCacheKey(js)
	if i.ssrCache != nil {
		if cached, ok := i.ssrCache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	reqHeader := map[string]string{
		"Content-Type": "application/json",
	}
	if len(i.ssrConfig.Headers) > 0 {
		for key, value := range i.ssrConfig.Headers {
			reqHeader[key] = value
		}
	}

	var reqCtx context.Context
	reqCtx = ctx
	var cancel context.CancelFunc
	if i.ssrConfig.Timeout > 0 {
		reqCtx, cancel = context.WithTimeout(reqCtx, i.ssrConfig.Timeout)
	}
	if cancel != nil {
		defer cancel()
	}

	var statusCode int
	var body []byte
	maxRetries := i.ssrConfig.MaxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		statusCode, body, err = i.ssrClient.Post(reqCtx, i.ssrConfig.URL, js, reqHeader)
		if err == nil && !shouldRetrySSRStatus(statusCode, i.ssrConfig.RetryStatuses) {
			break
		}
		if reqCtx.Err() != nil {
			break
		}
		if attempt < maxRetries {
			i.logger.WarnContext(
				ctx, "SSR retrying request",
				"attempt", attempt+1,
				"url", i.ssrConfig.URL,
				"status", statusCode,
				"error", err,
			)
			if err := sleeper(reqCtx, i.ssrConfig.RetryDelay); err != nil {
				return nil, fmt.Errorf("sleeper failed: %w", err)
			}
		}
	}

	if err != nil {
		i.logger.ErrorContext(ctx, "SSR request failed", "error", err, "url", i.ssrConfig.URL)
		return nil, fmt.Errorf("error posting ssr: %w", err)
	}

	if statusCode >= 400 {
		i.logger.ErrorContext(ctx, "SSR response error", "status", statusCode, "url", i.ssrConfig.URL)
		return nil, ErrBadSsrStatusCode
	}

	ssr := new(SsrDTO)
	err = json.Unmarshal(body, ssr)
	if err != nil {
		i.logger.ErrorContext(ctx, "SSR unmarshal failed", "error", err)
		return nil, fmt.Errorf("error unmarshalling ssr: %w", err)
	}

	if i.ssrCache != nil {
		i.ssrCache.Set(cacheKey, ssr)
	}

	return ssr, nil
}

func (i *Inertia) initSSRCache() {
	if i.ssrConfig.CacheTTL <= 0 {
		i.ssrCache = nil
		return
	}
	maxEntries := i.ssrConfig.CacheMaxEntries
	if maxEntries <= 0 {
		maxEntries = 256
	}
	i.ssrCache = newSSRCache(i.ssrConfig.CacheTTL, maxEntries)
}

func normalizeSSRConfig(cfg SSRConfig) SSRConfig {
	if cfg.URL == "" {
		return SSRConfig{}
	}
	if cfg.DisableRetries {
		cfg.MaxRetries = 0
	} else if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultSSRMaxRetries
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = DefaultSSRRetryDelay
	}
	return cfg
}

func shouldRetrySSRStatus(statusCode int, retryStatuses []int) bool {
	if statusCode == 0 {
		return true
	}

	if len(retryStatuses) == 0 {
		return statusCode >= 500
	}

	for _, code := range retryStatuses {
		if statusCode == code {
			return true
		}
	}

	return false
}

func ssrCacheKey(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
