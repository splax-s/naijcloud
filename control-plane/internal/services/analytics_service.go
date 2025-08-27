package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/naijcloud/control-plane/internal/models"
)

type AnalyticsService struct {
	db *sql.DB
}

func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{
		db: db,
	}
}

// GetDomainAnalytics retrieves analytics for a specific domain
func (s *AnalyticsService) GetDomainAnalytics(domain string, startTime, endTime time.Time) (*models.Analytics, error) {
	query := `
		SELECT 
			d.domain,
			COUNT(*) as total_requests,
			COALESCE(AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100, 0) as cache_hit_ratio,
			COALESCE(SUM(rl.bytes_sent), 0) as total_bytes_sent,
			COALESCE(AVG(rl.response_time_ms), 0) as avg_response_time,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p50_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p95_response_time,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p99_response_time
		FROM domains d
		LEFT JOIN request_logs rl ON d.id = rl.domain_id 
			AND rl.request_time >= $2 
			AND rl.request_time <= $3
		WHERE d.domain = $1
		GROUP BY d.domain
	`

	analytics := &models.Analytics{}
	var totalBytesSent int64

	err := s.db.QueryRow(query, domain, startTime, endTime).Scan(
		&analytics.Domain,
		&analytics.TotalRequests,
		&analytics.CacheHitRatio,
		&totalBytesSent,
		&analytics.AvgResponseTime,
		&analytics.P50ResponseTime,
		&analytics.P95ResponseTime,
		&analytics.P99ResponseTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.Analytics{Domain: domain}, nil
		}
		return nil, fmt.Errorf("failed to get domain analytics: %w", err)
	}

	// Calculate bandwidth saved (estimate: cache hits save 80% of bandwidth)
	if analytics.CacheHitRatio > 0 {
		analytics.BandwidthSaved = int64(float64(totalBytesSent) * (analytics.CacheHitRatio / 100) * 0.8)
	}

	return analytics, nil
}

// GetTopPaths returns the most requested paths for a domain
func (s *AnalyticsService) GetTopPaths(domain string, startTime, endTime time.Time, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			rl.path,
			COUNT(*) as request_count,
			COALESCE(AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100, 0) as cache_hit_ratio,
			COALESCE(AVG(rl.response_time_ms), 0) as avg_response_time
		FROM request_logs rl
		JOIN domains d ON rl.domain_id = d.id
		WHERE d.domain = $1 
			AND rl.request_time >= $2 
			AND rl.request_time <= $3
		GROUP BY rl.path
		ORDER BY request_count DESC
		LIMIT $4
	`

	rows, err := s.db.Query(query, domain, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top paths: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var path string
		var requestCount int64
		var cacheHitRatio, avgResponseTime float64

		err := rows.Scan(&path, &requestCount, &cacheHitRatio, &avgResponseTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top path: %w", err)
		}

		results = append(results, map[string]interface{}{
			"path":            path,
			"request_count":   requestCount,
			"cache_hit_ratio": cacheHitRatio,
			"avg_response_time": avgResponseTime,
		})
	}

	return results, nil
}

// GetRequestsOverTime returns request counts over time periods
func (s *AnalyticsService) GetRequestsOverTime(domain string, startTime, endTime time.Time, interval string) ([]map[string]interface{}, error) {
	// Validate interval
	validIntervals := map[string]bool{
		"1 hour": true, "1 day": true, "1 week": true,
	}
	if !validIntervals[interval] {
		interval = "1 hour"
	}

	query := `
		SELECT 
			DATE_TRUNC($4, rl.request_time) as time_bucket,
			COUNT(*) as request_count,
			COALESCE(AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100, 0) as cache_hit_ratio
		FROM request_logs rl
		JOIN domains d ON rl.domain_id = d.id
		WHERE d.domain = $1 
			AND rl.request_time >= $2 
			AND rl.request_time <= $3
		GROUP BY time_bucket
		ORDER BY time_bucket
	`

	rows, err := s.db.Query(query, domain, startTime, endTime, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get requests over time: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timeBucket time.Time
		var requestCount int64
		var cacheHitRatio float64

		err := rows.Scan(&timeBucket, &requestCount, &cacheHitRatio)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time series data: %w", err)
		}

		results = append(results, map[string]interface{}{
			"time":            timeBucket,
			"request_count":   requestCount,
			"cache_hit_ratio": cacheHitRatio,
		})
	}

	return results, nil
}

// LogRequest logs a request to the analytics database
func (s *AnalyticsService) LogRequest(log *models.RequestLog) error {
	query := `
		INSERT INTO request_logs 
		(id, domain_id, edge_id, request_time, method, path, status_code, response_time_ms, 
		 bytes_sent, cache_status, client_ip, user_agent, referer)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := s.db.Exec(query,
		log.ID, log.DomainID, log.EdgeID, log.RequestTime, log.Method, log.Path,
		log.StatusCode, log.ResponseTimeMs, log.BytesSent, log.CacheStatus,
		log.ClientIP, log.UserAgent, log.Referer,
	)
	if err != nil {
		return fmt.Errorf("failed to log request: %w", err)
	}

	return nil
}

// Organization-scoped analytics methods

// GetOrganizationOverview retrieves overview analytics for an organization
func (s *AnalyticsService) GetOrganizationOverview(orgID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	// Total requests across all domains for this organization
	var totalRequests int64
	var totalBandwidth int64
	var avgResponseTime float64
	var avgCacheHitRatio float64

	query := `
		SELECT 
			COUNT(*) as total_requests,
			COALESCE(SUM(rl.bytes_sent), 0) as total_bandwidth,
			COALESCE(AVG(rl.response_time_ms), 0) as avg_response_time,
			COALESCE(AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100, 0) as cache_hit_ratio
		FROM request_logs rl
		JOIN domains d ON d.id = rl.domain_id
		WHERE d.organization_id = $1 
		AND rl.request_time BETWEEN $2 AND $3
	`

	err := s.db.QueryRow(query, orgID, startTime, endTime).Scan(
		&totalRequests, &totalBandwidth, &avgResponseTime, &avgCacheHitRatio,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get organization overview: %w", err)
	}

	// Count domains
	var domainCount int
	domainQuery := `SELECT COUNT(*) FROM domains WHERE organization_id = $1`
	err = s.db.QueryRow(domainQuery, orgID).Scan(&domainCount)
	if err != nil {
		domainCount = 0
	}

	// Count edge nodes
	var edgeCount int
	edgeQuery := `SELECT COUNT(*) FROM edges WHERE organization_id = $1 OR organization_id IS NULL`
	err = s.db.QueryRow(edgeQuery, orgID).Scan(&edgeCount)
	if err != nil {
		edgeCount = 0
	}

	return map[string]interface{}{
		"total_requests":    totalRequests,
		"total_bandwidth":   totalBandwidth,
		"avg_response_time": avgResponseTime,
		"cache_hit_ratio":   avgCacheHitRatio,
		"domain_count":      domainCount,
		"edge_count":        edgeCount,
		"period_start":      startTime.Format(time.RFC3339),
		"period_end":        endTime.Format(time.RFC3339),
	}, nil
}

// GetOrganizationDomainAnalytics retrieves analytics for a specific domain within an organization
func (s *AnalyticsService) GetOrganizationDomainAnalytics(orgID, domain string, startTime, endTime time.Time) (*models.Analytics, error) {
	query := `
		SELECT 
			d.domain,
			COUNT(*) as total_requests,
			COALESCE(AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100, 0) as cache_hit_ratio,
			COALESCE(SUM(rl.bytes_sent), 0) as total_bytes_sent,
			COALESCE(AVG(rl.response_time_ms), 0) as avg_response_time,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p50_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p95_response_time,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY rl.response_time_ms), 0) as p99_response_time
		FROM domains d
		LEFT JOIN request_logs rl ON d.id = rl.domain_id 
			AND rl.request_time BETWEEN $3 AND $4
		WHERE d.organization_id = $1 AND d.domain = $2
		GROUP BY d.domain
	`

	analytics := &models.Analytics{}
	var totalBytesSent int64

	err := s.db.QueryRow(query, orgID, domain, startTime, endTime).Scan(
		&analytics.Domain,
		&analytics.TotalRequests,
		&analytics.CacheHitRatio,
		&totalBytesSent,
		&analytics.AvgResponseTime,
		&analytics.P50ResponseTime,
		&analytics.P95ResponseTime,
		&analytics.P99ResponseTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found or no data available")
		}
		return nil, fmt.Errorf("failed to get domain analytics: %w", err)
	}

	// Calculate bandwidth saved (assuming 50% cache hit ratio baseline)
	analytics.BandwidthSaved = int64(float64(totalBytesSent) * (analytics.CacheHitRatio / 100.0) * 0.5)

	return analytics, nil
}

// GetOrganizationUsageStats retrieves usage statistics for an organization
func (s *AnalyticsService) GetOrganizationUsageStats(orgID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	// Daily usage breakdown
	dailyQuery := `
		SELECT 
			DATE(rl.request_time) as date,
			COUNT(*) as requests,
			SUM(rl.bytes_sent) as bandwidth,
			AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100 as cache_hit_ratio
		FROM request_logs rl
		JOIN domains d ON d.id = rl.domain_id
		WHERE d.organization_id = $1 
		AND rl.request_time BETWEEN $2 AND $3
		GROUP BY DATE(rl.request_time)
		ORDER BY date DESC
		LIMIT 30
	`

	rows, err := s.db.Query(dailyQuery, orgID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}
	defer rows.Close()

	var dailyStats []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var requests int64
		var bandwidth int64
		var cacheHitRatio float64

		err := rows.Scan(&date, &requests, &bandwidth, &cacheHitRatio)
		if err != nil {
			continue
		}

		dailyStats = append(dailyStats, map[string]interface{}{
			"date":            date.Format("2006-01-02"),
			"requests":        requests,
			"bandwidth":       bandwidth,
			"cache_hit_ratio": cacheHitRatio,
		})
	}

	// Top domains by requests
	topDomainsQuery := `
		SELECT 
			d.domain,
			COUNT(*) as requests,
			SUM(rl.bytes_sent) as bandwidth
		FROM request_logs rl
		JOIN domains d ON d.id = rl.domain_id
		WHERE d.organization_id = $1 
		AND rl.request_time BETWEEN $2 AND $3
		GROUP BY d.domain
		ORDER BY requests DESC
		LIMIT 10
	`

	rows, err = s.db.Query(topDomainsQuery, orgID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get top domains: %w", err)
	}
	defer rows.Close()

	var topDomains []map[string]interface{}
	for rows.Next() {
		var domain string
		var requests int64
		var bandwidth int64

		err := rows.Scan(&domain, &requests, &bandwidth)
		if err != nil {
			continue
		}

		topDomains = append(topDomains, map[string]interface{}{
			"domain":    domain,
			"requests":  requests,
			"bandwidth": bandwidth,
		})
	}

	return map[string]interface{}{
		"daily_stats":  dailyStats,
		"top_domains":  topDomains,
		"period_start": startTime.Format(time.RFC3339),
		"period_end":   endTime.Format(time.RFC3339),
	}, nil
}

// GetOrganizationPerformanceMetrics retrieves performance metrics for an organization
func (s *AnalyticsService) GetOrganizationPerformanceMetrics(orgID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	query := `
		SELECT 
			AVG(rl.response_time_ms) as avg_response_time,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rl.response_time_ms) as p50_response_time,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY rl.response_time_ms) as p95_response_time,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY rl.response_time_ms) as p99_response_time,
			COUNT(CASE WHEN rl.status_code >= 400 THEN 1 END) as error_count,
			COUNT(*) as total_requests,
			AVG(CASE WHEN rl.cache_status = 'hit' THEN 1.0 ELSE 0.0 END) * 100 as cache_hit_ratio
		FROM request_logs rl
		JOIN domains d ON d.id = rl.domain_id
		WHERE d.organization_id = $1 
		AND rl.request_time BETWEEN $2 AND $3
	`

	var avgResponseTime, p50, p95, p99, cacheHitRatio float64
	var errorCount, totalRequests int64

	err := s.db.QueryRow(query, orgID, startTime, endTime).Scan(
		&avgResponseTime, &p50, &p95, &p99, &errorCount, &totalRequests, &cacheHitRatio,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get performance metrics: %w", err)
	}

	errorRate := float64(0)
	if totalRequests > 0 {
		errorRate = (float64(errorCount) / float64(totalRequests)) * 100
	}

	return map[string]interface{}{
		"avg_response_time": avgResponseTime,
		"p50_response_time": p50,
		"p95_response_time": p95,
		"p99_response_time": p99,
		"error_rate":        errorRate,
		"cache_hit_ratio":   cacheHitRatio,
		"total_requests":    totalRequests,
		"error_count":       errorCount,
		"period_start":      startTime.Format(time.RFC3339),
		"period_end":        endTime.Format(time.RFC3339),
	}, nil
}
