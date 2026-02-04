package azuread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/user/azure2aws/internal/provider"
	"github.com/user/azure2aws/internal/prompter"
)

// processConvergedTFA handles MFA (Two-Factor Authentication)
func (c *Client) processConvergedTFA(res *http.Response, resBodyStr string, creds *provider.LoginCredentials) (*http.Response, error) {
	var convergedResp ConvergedResponse
	if err := c.unmarshalEmbeddedJSON(resBodyStr, &convergedResp); err != nil {
		return nil, fmt.Errorf("failed to parse ConvergedTFA response: %w", err)
	}

	mfas := convergedResp.ArrUserProofs

	// If there's an option to skip MFA registration, use it
	if convergedResp.URLSkipMfaRegistration != "" {
		return c.httpClient.Get(convergedResp.URLSkipMfaRegistration)
	}

	// Process MFA if available
	if len(mfas) > 0 {
		return c.processMFA(mfas, &convergedResp, creds)
	}

	return res, nil
}

// processMFA handles the MFA flow
func (c *Client) processMFA(mfas []UserProof, convergedResp *ConvergedResponse, creds *provider.LoginCredentials) (*http.Response, error) {
	if len(mfas) == 0 {
		return nil, fmt.Errorf("no MFA methods available")
	}

	// Begin MFA authentication
	mfaResp, err := c.processMFABeginAuth(mfas, convergedResp)
	if err != nil {
		return nil, fmt.Errorf("MFA BeginAuth failed: %w", err)
	}

	// MFA polling loop
	for i := 0; ; i++ {
		mfaReq := MFARequest{
			AuthMethodID: mfaResp.AuthMethodID,
			Method:       "EndAuth",
			Ctx:          mfaResp.Ctx,
			FlowToken:    mfaResp.FlowToken,
			SessionID:    mfaResp.SessionID,
		}

		// Handle OTP-based MFA methods
		if mfaReq.AuthMethodID == MFAPhoneAppOTP || mfaReq.AuthMethodID == MFAOneWaySMS {
			if creds.MFAToken != "" {
				mfaReq.AdditionalAuthData = creds.MFAToken
			} else {
				verifyCode, err := prompter.String("Enter verification code", "")
				if err != nil {
					return nil, fmt.Errorf("failed to read verification code: %w", err)
				}
				mfaReq.AdditionalAuthData = verifyCode
			}
		}

		// Handle push notification on first iteration
		if mfaReq.AuthMethodID == MFAPhoneAppNotification && i == 0 {
			if mfaResp.Entropy == 0 {
				fmt.Println("Phone approval required.")
			} else {
				fmt.Printf("Phone approval required. Number match: %d\n", mfaResp.Entropy)
			}
		}

		// End MFA authentication
		mfaResp, err = c.processMFAEndAuth(mfaReq, convergedResp)
		if err != nil {
			return nil, fmt.Errorf("MFA EndAuth failed: %w", err)
		}

		if mfaResp.ErrCode != 0 {
			return nil, fmt.Errorf("MFA error %d: %v", mfaResp.ErrCode, mfaResp.Message)
		}

		if mfaResp.Success {
			break
		}

		if !mfaResp.Retry {
			break
		}

		// Wait before polling again
		if interval, ok := convergedResp.OPerAuthPollingInterval[mfaResp.AuthMethodID]; ok {
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			time.Sleep(2 * time.Second) // Default polling interval
		}
	}

	if !mfaResp.Success {
		return nil, fmt.Errorf("MFA authentication failed")
	}

	// Complete MFA authentication
	return c.processMFAAuth(mfaResp, convergedResp)
}

// processMFABeginAuth initiates MFA authentication
func (c *Client) processMFABeginAuth(mfas []UserProof, convergedResp *ConvergedResponse) (*MFAResponse, error) {
	// Select MFA method (prefer default, otherwise first available)
	mfa := mfas[0]
	for _, v := range mfas {
		if v.IsDefault {
			mfa = v
			break
		}
	}

	mfaReq := MFARequest{
		AuthMethodID: mfa.AuthMethodID,
		Method:       "BeginAuth",
		Ctx:          convergedResp.SCtx,
		FlowToken:    convergedResp.SFT,
	}

	mfaReqJSON, err := json.Marshal(mfaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MFA request: %w", err)
	}

	req, err := http.NewRequest("POST", convergedResp.URLBeginAuth, strings.NewReader(string(mfaReqJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create MFA BeginAuth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MFA BeginAuth request failed: %w", err)
	}
	defer res.Body.Close()

	var mfaResp MFAResponse
	if err := json.NewDecoder(res.Body).Decode(&mfaResp); err != nil {
		return nil, fmt.Errorf("failed to decode MFA BeginAuth response: %w", err)
	}

	if !mfaResp.Success {
		return nil, fmt.Errorf("MFA BeginAuth failed: %v", mfaResp.Message)
	}

	return &mfaResp, nil
}

// processMFAEndAuth completes MFA authentication
func (c *Client) processMFAEndAuth(mfaReq MFARequest, convergedResp *ConvergedResponse) (*MFAResponse, error) {
	mfaReqJSON, err := json.Marshal(mfaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MFA request: %w", err)
	}

	req, err := http.NewRequest("POST", convergedResp.URLEndAuth, strings.NewReader(string(mfaReqJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create MFA EndAuth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MFA EndAuth request failed: %w", err)
	}
	defer res.Body.Close()

	var mfaResp MFAResponse
	if err := json.NewDecoder(res.Body).Decode(&mfaResp); err != nil {
		return nil, fmt.Errorf("failed to decode MFA EndAuth response: %w", err)
	}

	return &mfaResp, nil
}

// processMFAAuth completes the MFA flow and continues authentication
func (c *Client) processMFAAuth(mfaResp *MFAResponse, convergedResp *ConvergedResponse) (*http.Response, error) {
	formValues := url.Values{}
	formValues.Set("request", mfaResp.Ctx)
	formValues.Set("mfaAuthMethod", mfaResp.AuthMethodID)
	formValues.Set("canary", convergedResp.Canary)
	formValues.Set("login", convergedResp.SPOSTUsername)
	formValues.Set(convergedResp.SFTName, mfaResp.FlowToken)

	req, err := http.NewRequest("POST", convergedResp.URLPost, strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create MFA completion request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.httpClient.Do(req)
}
