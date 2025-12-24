package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

// ConfigureContextFunc initializes the API client once and stores it in `meta`.
// All resources and data sources retrieve it via `meta.(*client.Client)`.
//
// Token is required either via provider configuration or OCP_TOKEN environment variable.
type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

// New creates a Client configured for the given endpoint and token.
// If insecure is true, TLS certificate verification is skipped.
func New(endpoint, token string, insecure bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure, // When true, skip TLS certificate verification.
		},
	}

	return &Client{
		endpoint: endpoint,
		token:    token,
		http:     &http.Client{Transport: transport},
	}
}

// gqlRequest is the JSON envelope for GraphQL requests.
type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// gqlResponse is the JSON envelope for GraphQL responses.
type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Do executes a GraphQL query or mutation and unmarshals the "data" field into `into`.
//
// If the response contains GraphQL errors, the first error is returned as Go error.
// If `into` is nil, the "data" payload is ignored (useful for mutations where only success matters).
func (c *Client) Do(query string, variables map[string]interface{}, into interface{}) error {
	reqBody, err := json.Marshal(gqlRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var gqlResp gqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return err
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	if into != nil {
		return json.Unmarshal(gqlResp.Data, into)
	}

	return nil
}
