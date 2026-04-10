package homeassistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

// NewTestClient creates a client with a custom HTTP client, for testing.
func NewTestClient(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) SetState(entityID string, sensor SensorState) error {
	body, err := json.Marshal(sensor)
	if err != nil {
		return fmt.Errorf("marshal sensor state: %w", err)
	}

	url := fmt.Sprintf("%s/api/states/%s", c.baseURL, entityID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HA API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
