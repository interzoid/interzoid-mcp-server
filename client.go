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
// x402 MODE (default):
//   When INTERZOID_API_KEY is not set, the request is sent WITHOUT the
//   "license" query parameter. This triggers the x402 payment flow — the
//   API server responds with 402 Payment Required, and the calling agent
//   or client is responsible for handling the payment negotiation.
//
// API KEY MODE:
//   When INTERZOID_API_KEY is set, it is passed as the "license" query
//   parameter for direct authenticated access (bypassing x402).
func callInterzoidAPI(apiKey string, endpoint string, params map[string]string) (map[string]interface{}, error) {
	u, err := url.Parse(interzoidBaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	q := u.Query()

	// Only include license key if provided — omitting it triggers x402
	if apiKey != "" {
		q.Set("license", apiKey)
	}

	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := httpClient.Get(u.String())
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
