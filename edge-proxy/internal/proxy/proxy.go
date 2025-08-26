package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/naijcloud/edge-proxy/internal/cache"
	"github.com/sirupsen/logrus"
)

type ProxyService struct {
	httpClient  *http.Client
	cache       cache.Cache
	defaultTTL  time.Duration
	maxBodySize int64
}

type ProxyConfig struct {
	DefaultTTL       time.Duration
	MaxBodySize      int64
	ConnectTimeout   time.Duration
	ResponseTimeout  time.Duration
	IdleConnTimeout  time.Duration
	MaxIdleConns     int
	MaxIdleConnsHost int
}

func NewProxyService(cache cache.Cache, config ProxyConfig) *ProxyService {
	// Configure HTTP client with timeouts
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsHost,
		IdleConnTimeout:     config.IdleConnTimeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.ResponseTimeout,
	}

	return &ProxyService{
		httpClient:  client,
		cache:       cache,
		defaultTTL:  config.DefaultTTL,
		maxBodySize: config.MaxBodySize,
	}
}

func (p *ProxyService) ServeHTTP(w http.ResponseWriter, r *http.Request, originURL string) {
	ctx := r.Context()

	// Generate cache key
	cacheKey := cache.GenerateCacheKey(r)

	// Try to serve from cache first
	if entry, found := p.cache.Get(ctx, cacheKey); found {
		p.serveCachedResponse(w, r, entry)
		return
	}

	// Cache miss - fetch from origin
	p.fetchAndServe(w, r, originURL, cacheKey)
}

func (p *ProxyService) serveCachedResponse(w http.ResponseWriter, r *http.Request, entry *cache.CacheEntry) {
	// Copy headers from cache
	for name, values := range entry.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Add cache status header
	w.Header().Set("X-Cache-Status", "HIT")
	w.Header().Set("X-Cache-Date", entry.CachedAt.Format(time.RFC3339))

	// Set status code and write body
	w.WriteHeader(entry.StatusCode)
	w.Write(entry.Body)

	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"cache_key":   cache.GenerateCacheKey(r),
		"status":      "HIT",
		"status_code": entry.StatusCode,
		"size":        len(entry.Body),
	}).Info("Cache hit")
}

func (p *ProxyService) fetchAndServe(w http.ResponseWriter, r *http.Request, originURL, cacheKey string) {
	ctx := r.Context()

	// Parse origin URL
	origin, err := url.Parse(originURL)
	if err != nil {
		http.Error(w, "Invalid origin URL", http.StatusBadGateway)
		return
	}

	// Create proxy request
	proxyReq, err := p.createProxyRequest(r, origin)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusBadGateway)
		return
	}

	// Execute request
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		logrus.WithError(err).WithField("origin", originURL).Error("Failed to fetch from origin")
		http.Error(w, "Failed to fetch from origin", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := p.readResponseBody(resp)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusBadGateway)
		return
	}

	// Copy response headers (excluding hop-by-hop headers)
	p.copyResponseHeaders(resp, w)

	// Add cache status
	w.Header().Set("X-Cache-Status", "MISS")

	// Write response
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	// Cache the response if it's cacheable
	if cache.IsCacheable(r, resp) {
		ttl := p.determineTTL(resp)
		entry := &cache.CacheEntry{
			StatusCode: resp.StatusCode,
			Headers:    make(http.Header),
			Body:       body,
			CachedAt:   time.Now(),
			TTL:        ttl,
		}

		// Copy cacheable headers
		for name, values := range resp.Header {
			if p.isCacheableHeader(name) {
				entry.Headers[name] = values
			}
		}

		if err := p.cache.Set(ctx, cacheKey, entry); err != nil {
			logrus.WithError(err).Warn("Failed to cache response")
		}

		logrus.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"cache_key":   cacheKey,
			"status":      "MISS",
			"status_code": resp.StatusCode,
			"size":        len(body),
			"ttl":         ttl.Seconds(),
		}).Info("Cache miss - response cached")
	} else {
		logrus.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      "MISS",
			"status_code": resp.StatusCode,
			"size":        len(body),
			"cacheable":   false,
		}).Info("Cache miss - response not cached")
	}
}

func (p *ProxyService) createProxyRequest(r *http.Request, origin *url.URL) (*http.Request, error) {
	// Create new URL with origin host but original path and query
	proxyURL := &url.URL{
		Scheme:   origin.Scheme,
		Host:     origin.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	// Create new request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers (excluding hop-by-hop headers)
	for name, values := range r.Header {
		if !p.isHopByHopHeader(name) {
			for _, value := range values {
				proxyReq.Header.Add(name, value)
			}
		}
	}

	// Set/override some headers
	proxyReq.Header.Set("Host", origin.Host)
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyReq.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
	if r.URL.Scheme == "" {
		proxyReq.Header.Set("X-Forwarded-Proto", "http")
	}

	return proxyReq, nil
}

func (p *ProxyService) readResponseBody(resp *http.Response) ([]byte, error) {
	// Check content length
	if resp.ContentLength > p.maxBodySize {
		return nil, fmt.Errorf("response body too large: %d bytes", resp.ContentLength)
	}

	// Use LimitReader to prevent reading too much data
	limitedReader := io.LimitReader(resp.Body, p.maxBodySize)

	return io.ReadAll(limitedReader)
}

func (p *ProxyService) copyResponseHeaders(resp *http.Response, w http.ResponseWriter) {
	for name, values := range resp.Header {
		if !p.isHopByHopHeader(name) {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
	}
}

func (p *ProxyService) determineTTL(resp *http.Response) time.Duration {
	// Check Cache-Control header
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		if maxAge := p.extractMaxAge(cacheControl); maxAge > 0 {
			return time.Duration(maxAge) * time.Second
		}
	}

	// Check Expires header
	if expires := resp.Header.Get("Expires"); expires != "" {
		if expTime, err := time.Parse(time.RFC1123, expires); err == nil {
			if ttl := time.Until(expTime); ttl > 0 {
				return ttl
			}
		}
	}

	// Use default TTL
	return p.defaultTTL
}

func (p *ProxyService) extractMaxAge(cacheControl string) int {
	parts := strings.Split(strings.ToLower(cacheControl), ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			if maxAge, err := strconv.Atoi(part[8:]); err == nil {
				return maxAge
			}
		}
	}
	return 0
}

func (p *ProxyService) isHopByHopHeader(name string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	name = strings.ToLower(name)
	for _, header := range hopByHopHeaders {
		if strings.ToLower(header) == name {
			return true
		}
	}
	return false
}

func (p *ProxyService) isCacheableHeader(name string) bool {
	// Don't cache these headers
	nonCacheableHeaders := []string{
		"Set-Cookie",
		"Authorization",
		"Proxy-Authorization",
		"Date",
		"Server",
	}

	name = strings.ToLower(name)
	for _, header := range nonCacheableHeaders {
		if strings.ToLower(header) == name {
			return false
		}
	}
	return true
}

// PurgeCache removes cached content for specific paths
func (p *ProxyService) PurgeCache(ctx context.Context, domain string, paths []string) error {
	// Common header variations to purge
	headerVariations := []map[string]string{
		{}, // No headers
		{"Accept": "*/*"},
		{"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		{"Accept": "application/json"},
		{"Accept-Encoding": "gzip"},
		{"Accept": "*/*", "Accept-Encoding": "gzip"},
	}

	for _, path := range paths {
		purgedCount := 0

		// For exact paths, try different header variations
		for _, headers := range headerVariations {
			req := &http.Request{
				Method: "GET",
				Host:   domain,
				URL:    &url.URL{Path: path},
				Header: make(http.Header),
			}

			for key, value := range headers {
				req.Header.Set(key, value)
			}

			cacheKey := cache.GenerateCacheKey(req)
			if err := p.cache.Delete(ctx, cacheKey); err == nil {
				purgedCount++
				logrus.WithFields(logrus.Fields{
					"cache_key": cacheKey,
					"domain":    domain,
					"path":      path,
				}).Info("Cache entry purged")
			}
		}

		if purgedCount == 0 {
			logrus.WithFields(logrus.Fields{
				"domain": domain,
				"path":   path,
			}).Warn("No cache entries found to purge")
		}
	}

	return nil
}
