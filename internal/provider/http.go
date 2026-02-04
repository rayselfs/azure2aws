package provider

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"runtime"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	UserAgent = "azure2aws/1.0"
)

type HTTPClient struct {
	*http.Client
	skipVerify bool
}

type HTTPClientOptions struct {
	SkipVerify bool
	Timeout    time.Duration
}

func DefaultHTTPClientOptions() *HTTPClientOptions {
	return &HTTPClientOptions{
		SkipVerify: false,
		Timeout:    60 * time.Second,
	}
}

func NewHTTPClient(opts *HTTPClientOptions) (*HTTPClient, error) {
	if opts == nil {
		opts = DefaultHTTPClientOptions()
	}

	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.SkipVerify,
			MinVersion:         tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   opts.Timeout,
	}

	return &HTTPClient{
		Client:     client,
		skipVerify: opts.SkipVerify,
	}, nil
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s %s)", UserAgent, runtime.GOOS, runtime.GOARCH))
	return c.Client.Do(req)
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *HTTPClient) PostForm(url string, data io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

func (c *HTTPClient) DisableFollowRedirect() {
	c.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
}

func (c *HTTPClient) EnableFollowRedirect() {
	c.Client.CheckRedirect = nil
}

func (c *HTTPClient) ClearCookies() error {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return fmt.Errorf("failed to create new cookie jar: %w", err)
	}
	c.Client.Jar = jar
	return nil
}
