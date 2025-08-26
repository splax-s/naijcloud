package cache

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type CacheEntry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CachedAt   time.Time
	TTL        time.Duration
}

type Cache interface {
	Get(ctx context.Context, key string) (*CacheEntry, bool)
	Set(ctx context.Context, key string, entry *CacheEntry) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Size() int64
}

// MemoryCache implements in-memory caching with LRU eviction
type MemoryCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int64
	currSize int64
}

// RedisCache implements Redis-backed caching
type RedisCache struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(maxSizeBytes int64) *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSizeBytes,
	}
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(redisURL, keyPrefix string, defaultTTL time.Duration) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:     client,
		keyPrefix:  keyPrefix,
		defaultTTL: defaultTTL,
	}, nil
}

// MemoryCache implementation
func (m *MemoryCache) Get(ctx context.Context, key string) (*CacheEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.CachedAt) > entry.TTL {
		delete(m.entries, key)
		m.currSize -= m.entrySize(entry)
		return nil, false
	}

	return entry, true
}

func (m *MemoryCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entrySize := m.entrySize(entry)

	// Remove existing entry if it exists
	if existing, exists := m.entries[key]; exists {
		m.currSize -= m.entrySize(existing)
	}

	// Evict entries if necessary
	for m.currSize+entrySize > m.maxSize && len(m.entries) > 0 {
		m.evictOldest()
	}

	m.entries[key] = entry
	m.currSize += entrySize

	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.entries[key]; exists {
		delete(m.entries, key)
		m.currSize -= m.entrySize(entry)
	}

	return nil
}

func (m *MemoryCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*CacheEntry)
	m.currSize = 0

	return nil
}

func (m *MemoryCache) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currSize
}

func (m *MemoryCache) entrySize(entry *CacheEntry) int64 {
	size := int64(len(entry.Body))
	for name, values := range entry.Headers {
		size += int64(len(name))
		for _, value := range values {
			size += int64(len(value))
		}
	}
	return size + 64 // overhead for struct fields
}

func (m *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range m.entries {
		if oldestKey == "" || entry.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CachedAt
		}
	}

	if oldestKey != "" {
		if entry := m.entries[oldestKey]; entry != nil {
			m.currSize -= m.entrySize(entry)
		}
		delete(m.entries, oldestKey)
	}
}

// RedisCache implementation
func (r *RedisCache) Get(ctx context.Context, key string) (*CacheEntry, bool) {
	fullKey := r.keyPrefix + key

	result, err := r.client.HGetAll(ctx, fullKey).Result()
	if err != nil || len(result) == 0 {
		return nil, false
	}

	entry := &CacheEntry{}

	// Parse status code
	if sc, exists := result["status_code"]; exists {
		if statusCode, err := strconv.Atoi(sc); err == nil {
			entry.StatusCode = statusCode
		}
	}

	// Parse headers
	entry.Headers = make(http.Header)
	if headersStr, exists := result["headers"]; exists {
		lines := strings.Split(headersStr, "\n")
		for _, line := range lines {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				entry.Headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	// Parse body
	if body, exists := result["body"]; exists {
		entry.Body = []byte(body)
	}

	// Parse cached time
	if cachedAtStr, exists := result["cached_at"]; exists {
		if timestamp, err := strconv.ParseInt(cachedAtStr, 10, 64); err == nil {
			entry.CachedAt = time.Unix(timestamp, 0)
		}
	}

	// Parse TTL
	if ttlStr, exists := result["ttl"]; exists {
		if ttlSecs, err := strconv.Atoi(ttlStr); err == nil {
			entry.TTL = time.Duration(ttlSecs) * time.Second
		}
	}

	// Check if entry has expired
	if time.Since(entry.CachedAt) > entry.TTL {
		r.client.Del(ctx, fullKey)
		return nil, false
	}

	return entry, true
}

func (r *RedisCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	fullKey := r.keyPrefix + key

	// Serialize headers
	var headerLines []string
	for name, values := range entry.Headers {
		for _, value := range values {
			headerLines = append(headerLines, fmt.Sprintf("%s: %s", name, value))
		}
	}
	headersStr := strings.Join(headerLines, "\n")

	// Set cache entry with expiration
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, fullKey, map[string]interface{}{
		"status_code": entry.StatusCode,
		"headers":     headersStr,
		"body":        string(entry.Body),
		"cached_at":   entry.CachedAt.Unix(),
		"ttl":         int(entry.TTL.Seconds()),
	})
	pipe.Expire(ctx, fullKey, entry.TTL)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.keyPrefix + key
	return r.client.Del(ctx, fullKey).Err()
}

func (r *RedisCache) Clear(ctx context.Context) error {
	pattern := r.keyPrefix + "*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}

func (r *RedisCache) Size() int64 {
	// For Redis, we can't easily get exact size, so return key count
	ctx := context.Background()
	pattern := r.keyPrefix + "*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		logrus.WithError(err).Warn("Failed to get cache size")
		return 0
	}
	return int64(len(keys))
}

// GenerateCacheKey creates a cache key from request
func GenerateCacheKey(req *http.Request) string {
	var buf bytes.Buffer
	buf.WriteString(req.Method)
	buf.WriteByte(':')
	buf.WriteString(req.Host)
	buf.WriteString(req.URL.Path)
	if req.URL.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(req.URL.RawQuery)
	}

	// Include relevant headers that affect caching
	headers := []string{"Accept", "Accept-Encoding", "Authorization"}
	for _, header := range headers {
		if value := req.Header.Get(header); value != "" {
			buf.WriteByte('|')
			buf.WriteString(header)
			buf.WriteByte('=')
			buf.WriteString(value)
		}
	}

	return buf.String()
}

// IsCacheable determines if a response should be cached
func IsCacheable(req *http.Request, resp *http.Response) bool {
	// Don't cache non-GET requests
	if req.Method != "GET" && req.Method != "HEAD" {
		return false
	}

	// Don't cache responses with Set-Cookie headers
	if resp.Header.Get("Set-Cookie") != "" {
		return false
	}

	// Don't cache private responses
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		if strings.Contains(strings.ToLower(cacheControl), "private") ||
			strings.Contains(strings.ToLower(cacheControl), "no-cache") ||
			strings.Contains(strings.ToLower(cacheControl), "no-store") {
			return false
		}
	}

	// Only cache successful responses
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
