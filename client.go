package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "http://localhost:6333"

// Client is a Qdrant API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// ClientConfig contains configuration for creating a Qdrant client.
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// NewClient creates a new Qdrant client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Point represents a vector point in Qdrant.
type Point struct {
	ID      string            `json:"id"`
	Vector  []float64         `json:"vector"`
	Payload map[string]string `json:"payload,omitempty"`
}

// UpsertRequest is the request body for upserting points.
type UpsertRequest struct {
	Points []Point `json:"points"`
}

// SearchRequest is the request body for searching.
type SearchRequest struct {
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVector  bool      `json:"with_vector"`
}

// SearchResult represents a search result.
type SearchResult struct {
	ID      string            `json:"id"`
	Score   float64           `json:"score"`
	Payload map[string]string `json:"payload,omitempty"`
	Vector  []float64         `json:"vector,omitempty"`
}

// CollectionConfig defines collection configuration.
type CollectionConfig struct {
	Vectors VectorConfig `json:"vectors"`
}

// VectorConfig defines vector configuration.
type VectorConfig struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
}

// Upsert adds or updates points in a collection.
func (c *Client) Upsert(ctx context.Context, collection string, points []Point) error {
	req := UpsertRequest{Points: points}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/collections/%s/points?wait=true", c.baseURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Search performs a vector similarity search.
func (c *Client) Search(ctx context.Context, collection string, vector []float64, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	req := SearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Result, nil
}

// Delete removes points from a collection.
func (c *Client) Delete(ctx context.Context, collection string, ids []string) error {
	req := map[string][]string{"points": ids}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CreateCollection creates a new collection.
func (c *Client) CreateCollection(ctx context.Context, name string, vectorSize int, distance string) error {
	if distance == "" {
		distance = "Cosine"
	}

	req := CollectionConfig{
		Vectors: VectorConfig{
			Size:     vectorSize,
			Distance: distance,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
}
