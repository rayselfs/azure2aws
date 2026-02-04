package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	FederationEndpoint = "https://signin.aws.amazon.com/federation"
	ConsoleURL         = "https://console.aws.amazon.com/"
	Issuer             = "azure2aws"
)

type SigninTokenResponse struct {
	SigninToken string `json:"SigninToken"`
}

func GetFederatedLoginURL(creds *Credentials, service string) (string, error) {
	signinToken, err := getSigninToken(creds)
	if err != nil {
		return "", fmt.Errorf("failed to get signin token: %w", err)
	}

	destination := ConsoleURL
	if service != "" {
		destination = fmt.Sprintf("https://%s.console.aws.amazon.com/", service)
	}

	loginURL := fmt.Sprintf(
		"%s?Action=login&Issuer=%s&Destination=%s&SigninToken=%s",
		FederationEndpoint,
		url.QueryEscape(Issuer),
		url.QueryEscape(destination),
		url.QueryEscape(signinToken),
	)

	return loginURL, nil
}

func getSigninToken(creds *Credentials) (string, error) {
	sessionJSON, err := json.Marshal(map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	req, err := http.NewRequest("GET", FederationEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(sessionJSON))
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getSigninToken request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResp SigninTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.SigninToken == "" {
		return "", fmt.Errorf("signin token not found in response")
	}

	return tokenResp.SigninToken, nil
}
