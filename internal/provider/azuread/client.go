package azuread

import (
	"fmt"

	"github.com/user/azure2aws/internal/provider"
)

// Client handles Azure AD SAML authentication
type Client struct {
	httpClient *provider.HTTPClient
	baseURL    string
	appID      string
}

// ClientOptions contains configuration for the Azure AD client
type ClientOptions struct {
	URL        string // Azure AD base URL (e.g., https://account.activedirectory.windowsazure.com)
	AppID      string // Azure AD application ID
	SkipVerify bool   // Skip TLS certificate verification
}

// NewClient creates a new Azure AD authentication client
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if opts.AppID == "" {
		return nil, fmt.Errorf("AppID is required")
	}

	httpOpts := provider.DefaultHTTPClientOptions()
	httpOpts.SkipVerify = opts.SkipVerify

	httpClient, err := provider.NewHTTPClient(httpOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    opts.URL,
		appID:      opts.AppID,
	}, nil
}

// Authenticate performs Azure AD SAML authentication
// Returns the base64-encoded SAML assertion
func (c *Client) Authenticate(creds *provider.LoginCredentials) (string, error) {
	if creds == nil {
		return "", fmt.Errorf("credentials cannot be nil")
	}

	if creds.Username == "" {
		return "", fmt.Errorf("username is required")
	}

	if creds.Password == "" {
		return "", fmt.Errorf("password is required")
	}

	return c.authenticate(creds)
}
