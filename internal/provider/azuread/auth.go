package azuread

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/azure2aws/internal/provider"
)

// authenticate is the main authentication state machine
func (c *Client) authenticate(creds *provider.LoginCredentials) (string, error) {
	// Start the SAML flow
	startURL := fmt.Sprintf("%s/applications/redirecttofederatedapplication.aspx?Operation=LinkedSignIn&applicationId=%s",
		c.baseURL, c.appID)

	res, err := c.httpClient.Get(startURL)
	if err != nil {
		return "", fmt.Errorf("failed to start authentication: %w", err)
	}

	// Main authentication loop - state machine
	for {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}
		res.Body.Close()
		resBodyStr := string(resBody)

		// Reset body for potential re-reading
		res.Body = io.NopCloser(bytes.NewBuffer(resBody))

		switch {
		case strings.Contains(resBodyStr, "ConvergedSignIn"):
			res, err = c.processConvergedSignIn(res, resBodyStr, creds)
			if err != nil {
				return "", fmt.Errorf("ConvergedSignIn failed: %w", err)
			}

		case strings.Contains(resBodyStr, "ConvergedTFA"):
			res, err = c.processConvergedTFA(res, resBodyStr, creds)
			if err != nil {
				return "", fmt.Errorf("ConvergedTFA failed: %w", err)
			}

		case strings.Contains(resBodyStr, "KmsiInterrupt"):
			res, err = c.processKmsiInterrupt(res, resBodyStr)
			if err != nil {
				return "", fmt.Errorf("KmsiInterrupt failed: %w", err)
			}

		case strings.Contains(resBodyStr, "SAMLRequest"):
			res, err = c.processSAMLRequest(res, resBodyStr)
			if err != nil {
				return "", fmt.Errorf("SAMLRequest failed: %w", err)
			}

		case c.isHiddenForm(resBodyStr):
			if samlAssertion := c.getSAMLAssertion(resBodyStr); samlAssertion != "" {
				return samlAssertion, nil
			}
			res, err = c.reProcessForm(resBodyStr)
			if err != nil {
				return "", fmt.Errorf("form reprocessing failed: %w", err)
			}

		default:
			// Check for error in response
			if strings.Contains(resBodyStr, "sErrorCode") {
				var convergedResp ConvergedResponse
				if err := c.unmarshalEmbeddedJSON(resBodyStr, &convergedResp); err == nil {
					if convergedResp.SErrorCode != "" && convergedResp.SErrorCode != "50058" {
						return "", fmt.Errorf("authentication error: %s - %s", convergedResp.SErrorCode, convergedResp.SErrTxt)
					}
				}
			}
			return "", fmt.Errorf("reached unknown authentication state")
		}

		if err != nil {
			return "", err
		}
	}
}

// processConvergedSignIn handles the converged sign-in page
func (c *Client) processConvergedSignIn(res *http.Response, resBodyStr string, creds *provider.LoginCredentials) (*http.Response, error) {
	var convergedResp ConvergedResponse
	if err := c.unmarshalEmbeddedJSON(resBodyStr, &convergedResp); err != nil {
		return nil, fmt.Errorf("failed to parse ConvergedSignIn response: %w", err)
	}

	loginURL := c.fullURL(res, convergedResp.URLPost)
	refererURL := res.Request.URL.String()

	// Get credential type to check for federation
	credTypeResp, _, err := c.requestGetCredentialType(refererURL, creds, &convergedResp)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential type: %w", err)
	}

	// Check if federated authentication is needed
	if credTypeResp.Credentials.FederationRedirectURL != "" {
		return c.processFederatedAuth(credTypeResp.Credentials.FederationRedirectURL, creds)
	}

	// Process normal authentication
	return c.processAuthentication(loginURL, refererURL, creds, &convergedResp)
}

// requestGetCredentialType checks what type of credential the user needs
func (c *Client) requestGetCredentialType(refererURL string, creds *provider.LoginCredentials, convergedResp *ConvergedResponse) (*GetCredentialTypeResponse, *http.Response, error) {
	reqBody := GetCredentialTypeRequest{
		Username:            creds.Username,
		IsOtherIdpSupported: true,
		OriginalRequest:     convergedResp.SCtx,
		FlowToken:           convergedResp.SFT,
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", convergedResp.URLGetCredentialType, strings.NewReader(string(reqBodyJSON)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("canary", convergedResp.APICanary)
	req.Header.Set("client-request-id", convergedResp.CorrelationID)
	req.Header.Set("hpgact", fmt.Sprint(convergedResp.Hpgact))
	req.Header.Set("hpgid", fmt.Sprint(convergedResp.Hpgid))
	req.Header.Set("hpgrequestid", convergedResp.SessionID)
	req.Header.Set("Referer", refererURL)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}

	var credTypeResp GetCredentialTypeResponse
	if err := json.NewDecoder(res.Body).Decode(&credTypeResp); err != nil {
		return nil, res, fmt.Errorf("failed to decode response: %w", err)
	}

	return &credTypeResp, res, nil
}

// processAuthentication handles password authentication
func (c *Client) processAuthentication(loginURL, refererURL string, creds *provider.LoginCredentials, convergedResp *ConvergedResponse) (*http.Response, error) {
	// Check for login errors (50058 = user not signed in yet, which is expected)
	if convergedResp.SErrorCode != "" && convergedResp.SErrorCode != "50058" {
		return nil, fmt.Errorf("login error: %s - %s", convergedResp.SErrorCode, convergedResp.SErrTxt)
	}

	formValues := url.Values{}
	formValues.Set("canary", convergedResp.Canary)
	formValues.Set("hpgrequestid", convergedResp.SessionID)
	formValues.Set(convergedResp.SFTName, convergedResp.SFT)
	formValues.Set("ctx", convergedResp.SCtx)
	formValues.Set("login", creds.Username)
	formValues.Set("loginfmt", creds.Username)
	formValues.Set("passwd", creds.Password)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", refererURL)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}

	return res, nil
}

// processFederatedAuth handles ADFS federation
func (c *Client) processFederatedAuth(federationURL string, creds *provider.LoginCredentials) (*http.Response, error) {
	res, err := c.httpClient.Get(federationURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get federation URL: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read federation response: %w", err)
	}
	res.Body.Close()
	resBodyStr := string(resBody)

	formValues, formSubmitURL, err := c.parseFormData(resBodyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ADFS form: %w", err)
	}

	if formSubmitURL == "" {
		return nil, fmt.Errorf("ADFS form submit URL not found")
	}

	formValues.Set("UserName", creds.Username)
	formValues.Set("Password", creds.Password)
	formValues.Set("AuthMethod", "FormsAuthentication")

	req, err := http.NewRequest("POST", c.fullURL(res, formSubmitURL), strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create ADFS login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.httpClient.Do(req)
}

// processKmsiInterrupt handles the "Keep Me Signed In" page
func (c *Client) processKmsiInterrupt(res *http.Response, resBodyStr string) (*http.Response, error) {
	var convergedResp ConvergedResponse
	if err := c.unmarshalEmbeddedJSON(resBodyStr, &convergedResp); err != nil {
		return nil, fmt.Errorf("failed to parse KMSI response: %w", err)
	}

	formValues := url.Values{}
	formValues.Set(convergedResp.SFTName, convergedResp.SFT)
	formValues.Set("ctx", convergedResp.SCtx)
	formValues.Set("LoginOptions", "1") // Don't stay signed in

	req, err := http.NewRequest("POST", c.fullURL(res, convergedResp.URLPost), strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create KMSI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.httpClient.DisableFollowRedirect()
	newRes, err := c.httpClient.Do(req)
	c.httpClient.EnableFollowRedirect()

	if err != nil {
		return nil, fmt.Errorf("KMSI request failed: %w", err)
	}

	return newRes, nil
}

// processSAMLRequest handles SAML request forms
func (c *Client) processSAMLRequest(res *http.Response, resBodyStr string) (*http.Response, error) {
	formValues, formSubmitURL, err := c.parseFormData(resBodyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAML request form: %w", err)
	}

	if formSubmitURL == "" {
		return nil, fmt.Errorf("SAML request form URL not found")
	}

	req, err := http.NewRequest("POST", c.fullURL(res, formSubmitURL), strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create SAML request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.httpClient.Do(req)
}

// reProcessForm handles hidden form submissions
func (c *Client) reProcessForm(resBodyStr string) (*http.Response, error) {
	formValues, formSubmitURL, err := c.parseFormData(resBodyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	if formSubmitURL == "" {
		return nil, fmt.Errorf("form URL not found")
	}

	req, err := http.NewRequest("POST", formSubmitURL, strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create form request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.httpClient.Do(req)
}

// Helper methods

// fullURL constructs an absolute URL from a relative one
func (c *Client) fullURL(res *http.Response, relativeURL string) string {
	if strings.HasPrefix(relativeURL, "http") {
		return relativeURL
	}

	baseURL := res.Request.URL
	parsed, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	return baseURL.ResolveReference(parsed).String()
}

// unmarshalEmbeddedJSON extracts and parses $Config JSON from HTML
func (c *Client) unmarshalEmbeddedJSON(html string, v interface{}) error {
	re := regexp.MustCompile(`\$Config=({[^;]+});`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return fmt.Errorf("$Config not found in response")
	}

	return json.Unmarshal([]byte(matches[1]), v)
}

// isHiddenForm checks if the response contains a hidden form
func (c *Client) isHiddenForm(html string) bool {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return false
	}

	return doc.Find("form").Length() > 0 &&
		(doc.Find("input[type='hidden']").Length() > 0 || doc.Find("input[name='SAMLResponse']").Length() > 0)
}

// getSAMLAssertion extracts the SAML assertion from a form
func (c *Client) getSAMLAssertion(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	samlResponse, exists := doc.Find("input[name='SAMLResponse']").Attr("value")
	if exists {
		return samlResponse
	}

	return ""
}

// parseFormData extracts form fields and action URL from HTML
func (c *Client) parseFormData(html string) (url.Values, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	form := doc.Find("form").First()
	if form.Length() == 0 {
		return nil, "", fmt.Errorf("form not found")
	}

	action, _ := form.Attr("action")
	values := url.Values{}

	form.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, nameExists := s.Attr("name")
		value, _ := s.Attr("value")
		if nameExists && name != "" {
			values.Set(name, value)
		}
	})

	return values, action, nil
}
