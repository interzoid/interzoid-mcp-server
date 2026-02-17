package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	interzoidBaseURL = "https://api.interzoid.com"
	httpTimeout      = 30 * time.Second
)

var httpClient = &http.Client{Timeout: httpTimeout}

// callInterzoidAPI makes an HTTP GET request to the Interzoid API endpoint.
//
// Authentication priority (first match wins):
//   1. API key passed in from the connecting client's Authorization header
//      (remote HTTP transport — user provides their own key)
//   2. INTERZOID_API_KEY environment variable (local stdio transport)
//   3. No key — triggers x402 payment flow
//
// When an API key is available, it is sent as the "x-api-key" header to
// the Interzoid API (matching the existing API authentication convention).
// When no key is available, the request is sent without authentication,
// triggering a 402 Payment Required response for x402 payment negotiation.
func callInterzoidAPI(apiKey string, endpoint string, params map[string]string) (map[string]interface{}, error) {
	u, err := url.Parse(interzoidBaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send API key via x-api-key header (matching Interzoid API convention)
	// Omitting it triggers the x402 payment flow
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// In x402 mode, a 402 is expected — return the payment requirements
	// so the calling agent/client can handle the payment flow
	if resp.StatusCode == http.StatusPaymentRequired {
		var paymentReq map[string]interface{}
		if err := json.Unmarshal(body, &paymentReq); err != nil {
			return nil, fmt.Errorf("402 Payment Required: %s", string(body))
		}
		return map[string]interface{}{
			"status":              "payment_required",
			"x402":               true,
			"paymentRequirements": paymentReq,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}
