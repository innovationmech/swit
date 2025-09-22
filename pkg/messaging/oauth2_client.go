// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth2TokenResponse models a subset of RFC 6749 token response for client credentials.
type OAuth2TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

// FetchOAuth2ClientCredentialsToken performs a client_credentials grant request.
// It returns the access token string and its absolute expiration time (best-effort).
func FetchOAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret, tokenURL string, scopes []string) (string, time.Time, error) {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(tokenURL) == "" {
		return "", time.Time{}, NewConfigError("oauth2 client credentials require client_id, client_secret and token_url", nil)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create oauth2 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("oauth2 token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Avoid logging secrets; response body is generally safe to include
		return "", time.Time{}, fmt.Errorf("oauth2 token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tr OAuth2TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse oauth2 token response: %w", err)
	}
	if strings.TrimSpace(tr.AccessToken) == "" {
		return "", time.Time{}, errors.New("oauth2 token response missing access_token")
	}

	// Compute expiry if provided
	var expiresAt time.Time
	if tr.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tr.AccessToken, expiresAt, nil
}
