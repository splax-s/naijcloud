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
