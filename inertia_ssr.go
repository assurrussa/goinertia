package goinertia

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberclient "github.com/gofiber/fiber/v3/client"
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
	client *fiberclient.Client
}

func (c *defaultSSRClient) Reset() {
	c.client.Reset()
}

func (c *defaultSSRClient) Post(ctx context.Context, url string, body []byte, headers map[string]string) (int, []byte, error) {
	reqCfg := fiberclient.Config{
		Ctx:    ctx,
		Body:   body,
		Header: headers,
	}
	resp, err := c.client.Post(url, reqCfg)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Close()

	return resp.StatusCode(), resp.Body(), nil
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
		i.ssrClient = &defaultSSRClient{client: fiberclient.New()}
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

func (i *Inertia) processSSR(c fiber.Ctx, page *PageDTO) (*SsrDTO, error) {
	if !i.IsSSREnabled() {
		return nil, nil //nolint:nilnil // is need
	}

	var err error

	js, err := json.Marshal(page)
	if err != nil {
		i.logger.ErrorContext(c, "SSR marshal failed", "error", err)
		return nil, fmt.Errorf("error marshaling page: %w", err)
	}

	cacheKey := ssrCacheKey(js)
	if i.ssrCache != nil {
		if cached, ok := i.ssrCache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	reqHeader := map[string]string{
		fiber.HeaderContentType: fiber.MIMEApplicationJSON,
	}
	if len(i.ssrConfig.Headers) > 0 {
		for key, value := range i.ssrConfig.Headers {
			reqHeader[key] = value
		}
	}

	var reqCtx context.Context
	reqCtx = c.Context()
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
			if i.ssrClient != nil {
				i.ssrClient.Reset()
			}
			i.logger.WarnContext(
				c, "SSR retrying request",
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
		i.logger.ErrorContext(c, "SSR request failed", "error", err, "url", i.ssrConfig.URL)
		return nil, fmt.Errorf("error posting ssr: %w", err)
	}

	if statusCode >= 400 {
		i.logger.ErrorContext(c, "SSR response error", "status", statusCode, "url", i.ssrConfig.URL)
		return nil, ErrBadSsrStatusCode
	}

	ssr := new(SsrDTO)
	err = json.Unmarshal(body, ssr)
	if err != nil {
		i.logger.ErrorContext(c, "SSR unmarshal failed", "error", err)
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
