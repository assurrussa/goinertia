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
	DefaultSSRURL     = "http://127.0.0.1:13714"
	DefaultSSRTimeout = 3 * time.Second
)

type SSRConfig struct {
	URL             string
	Timeout         time.Duration
	Headers         map[string]string
	CacheTTL        time.Duration
	CacheMaxEntries int
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
	if i.ssrClient == nil {
		i.ssrClient = fiberclient.New()
	}
	i.initSSRCache()
}

func (i *Inertia) EnableSSRWithDefault() {
	i.EnableSSR(SSRConfig{
		URL:     DefaultSSRURL,
		Timeout: DefaultSSRTimeout,
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

	reqCfg := fiberclient.Config{
		Header: map[string]string{
			fiber.HeaderContentType: fiber.MIMEApplicationJSON,
		},
		Body:    js,
		Timeout: i.ssrConfig.Timeout,
	}
	if len(i.ssrConfig.Headers) > 0 {
		for key, value := range i.ssrConfig.Headers {
			reqCfg.Header[key] = value
		}
	}
	if c != nil {
		reqCtx := c.Context()
		var cancel context.CancelFunc
		if i.ssrConfig.Timeout > 0 {
			reqCtx, cancel = context.WithTimeout(reqCtx, i.ssrConfig.Timeout)
		}
		if cancel != nil {
			defer cancel()
		}
		reqCfg.Ctx = reqCtx
	}

	resp, err := i.ssrClient.Post(i.ssrConfig.URL, reqCfg)
	if err != nil {
		i.logger.ErrorContext(c, "SSR request failed", "error", err, "url", i.ssrConfig.URL)
		return nil, fmt.Errorf("error posting ssr: %w", err)
	}

	if resp.StatusCode() >= 400 {
		i.logger.ErrorContext(c, "SSR response error", "status", resp.StatusCode(), "url", i.ssrConfig.URL)
		return nil, ErrBadSsrStatusCode
	}

	ssr := new(SsrDTO)
	body := resp.Body()
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
	return cfg
}

func ssrCacheKey(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
