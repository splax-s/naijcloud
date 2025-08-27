package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ControlPlaneClient struct {
	baseURL    string
	httpClient *http.Client
	edgeID     uuid.UUID
	region     string
}

type EdgeRegistrationRequest struct {
	Region    string `json:"region"`
	IPAddress string `json:"ip_address"`
	Hostname  string `json:"hostname"`
	Capacity  int    `json:"capacity"`
}

type EdgeRegistrationResponse struct {
	ID        uuid.UUID `json:"id"`
	Region    string    `json:"region"`
	IPAddress string    `json:"ip_address"`
	Hostname  string    `json:"hostname"`
	Capacity  int       `json:"capacity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HeartbeatRequest struct {
	Status  string                 `json:"status"`
	Metrics map[string]interface{} `json:"metrics"`
}

type DomainResponse struct {
	ID        uuid.UUID `json:"id"`
	Domain    string    `json:"domain"`
	OriginURL string    `json:"origin_url"`
	CacheTTL  int       `json:"cache_ttl"`
	RateLimit int       `json:"rate_limit"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PurgeRequest struct {
	ID          uuid.UUID `json:"id"`
	DomainID    uuid.UUID `json:"domain_id"`
	Paths       []string  `json:"paths"`
	Status      string    `json:"status"`
	RequestedBy string    `json:"requested_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewControlPlaneClient(baseURL, region string) *ControlPlaneClient {
	return &ControlPlaneClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		region: region,
	}
}

func (c *ControlPlaneClient) RegisterEdge(ctx context.Context, ipAddress, hostname string, capacity int) (*EdgeRegistrationResponse, error) {
	req := EdgeRegistrationRequest{
		Region:    c.region,
		IPAddress: ipAddress,
		Hostname:  hostname,
		Capacity:  capacity,
	}

	var resp EdgeRegistrationResponse
	if err := c.makeRequest(ctx, "POST", "/api/v1/edges", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to register edge: %w", err)
	}

	c.edgeID = resp.ID
	logrus.WithFields(logrus.Fields{
		"edge_id":  resp.ID,
		"region":   resp.Region,
		"hostname": resp.Hostname,
	}).Info("Edge node registered with control plane")

	return &resp, nil
}

func (c *ControlPlaneClient) SendHeartbeat(ctx context.Context, status string, metrics map[string]interface{}) error {
	if c.edgeID == uuid.Nil {
		return fmt.Errorf("edge not registered")
	}

	req := HeartbeatRequest{
		Status:  status,
		Metrics: metrics,
	}

	endpoint := fmt.Sprintf("/api/v1/edges/%s/heartbeat", c.edgeID)
	return c.makeRequest(ctx, "POST", endpoint, req, nil)
}

func (c *ControlPlaneClient) GetDomain(ctx context.Context, domain string) (*DomainResponse, error) {
	var resp DomainResponse
	endpoint := fmt.Sprintf("/v1/domains/%s", domain)

	if err := c.makeRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get domain info: %w", err)
	}

	return &resp, nil
}

func (c *ControlPlaneClient) GetDomainByID(ctx context.Context, domainID uuid.UUID) (*DomainResponse, error) {
	var resp DomainResponse
	endpoint := fmt.Sprintf("/v1/domains/id/%s", domainID)

	if err := c.makeRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get domain info by ID: %w", err)
	}

	return &resp, nil
}

func (c *ControlPlaneClient) GetPendingPurges(ctx context.Context) ([]PurgeRequest, error) {
	if c.edgeID == uuid.Nil {
		return nil, fmt.Errorf("edge not registered")
	}

	// Control plane returns {"purges": [...]} so we need a wrapper struct
	var response struct {
		Purges []PurgeRequest `json:"purges"`
	}
	endpoint := fmt.Sprintf("/api/v1/edges/%s/purges", c.edgeID)

	if err := c.makeRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get pending purges: %w", err)
	}

	return response.Purges, nil
}

func (c *ControlPlaneClient) CompletePurge(ctx context.Context, purgeID uuid.UUID) error {
	if c.edgeID == uuid.Nil {
		return fmt.Errorf("edge not registered")
	}

	endpoint := fmt.Sprintf("/api/v1/edges/%s/purges/%s/complete", c.edgeID, purgeID)
	return c.makeRequest(ctx, "POST", endpoint, nil, nil)
}

func (c *ControlPlaneClient) makeRequest(ctx context.Context, method, endpoint string, reqBody interface{}, respBody interface{}) error {
	url := c.baseURL + endpoint

	var body bytes.Buffer
	if reqBody != nil {
		if err := json.NewEncoder(&body).Encode(reqBody); err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *ControlPlaneClient) GetEdgeID() uuid.UUID {
	return c.edgeID
}
