package tibber

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultAPIURL = "https://api.tibber.com/v1-beta/gql"

type Client struct {
	token      string
	apiURL     string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		apiURL:     defaultAPIURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) query(gql string, variables map[string]any) (*GraphQLResponse, error) {
	reqBody := GraphQLRequest{
		Query:     gql,
		Variables: variables,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result GraphQLResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return &result, nil
}

func (c *Client) GetHomes() ([]Home, error) {
	resp, err := c.query(HomesQuery, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data.Viewer.Homes, nil
}

func (c *Client) GetConsumption(last int) ([]Home, error) {
	vars := map[string]any{"last": last}
	resp, err := c.query(ConsumptionQuery, vars)
	if err != nil {
		return nil, err
	}
	return resp.Data.Viewer.Homes, nil
}

func (c *Client) GetPrices() ([]Home, error) {
	resp, err := c.query(PricesQuery, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data.Viewer.Homes, nil
}
