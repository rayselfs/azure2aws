package azuread

import "time"

// ConvergedResponse represents the response from Azure AD converged sign-in flow
type ConvergedResponse struct {
	URLGetCredentialType    string             `json:"urlGetCredentialType"`
	ArrUserProofs           []UserProof        `json:"arrUserProofs"`
	URLSkipMfaRegistration  string             `json:"urlSkipMfaRegistration"`
	OPerAuthPollingInterval map[string]float64 `json:"oPerAuthPollingInterval"`
	URLBeginAuth            string             `json:"urlBeginAuth"`
	URLEndAuth              string             `json:"urlEndAuth"`
	URLPost                 string             `json:"urlPost"`
	SErrorCode              string             `json:"sErrorCode"`
	SErrTxt                 string             `json:"sErrTxt"`
	SPOSTUsername           string             `json:"sPOST_Username"`
	SFT                     string             `json:"sFT"`
	SFTName                 string             `json:"sFTName"`
	SCtx                    string             `json:"sCtx"`
	Hpgact                  int                `json:"hpgact"`
	Hpgid                   int                `json:"hpgid"`
	Pgid                    string             `json:"pgid"`
	APICanary               string             `json:"apiCanary"`
	Canary                  string             `json:"canary"`
	CorrelationID           string             `json:"correlationId"`
	SessionID               string             `json:"sessionId"`
}

// GetCredentialTypeRequest is the request body for credential type detection
type GetCredentialTypeRequest struct {
	Username                       string `json:"username"`
	IsOtherIdpSupported            bool   `json:"isOtherIdpSupported"`
	CheckPhones                    bool   `json:"checkPhones"`
	IsRemoteNGCSupported           bool   `json:"isRemoteNGCSupported"`
	IsCookieBannerShown            bool   `json:"isCookieBannerShown"`
	IsFidoSupported                bool   `json:"isFidoSupported"`
	OriginalRequest                string `json:"originalRequest"`
	Country                        string `json:"country"`
	Forceotclogin                  bool   `json:"forceotclogin"`
	IsExternalFederationDisallowed bool   `json:"isExternalFederationDisallowed"`
	IsRemoteConnectSupported       bool   `json:"isRemoteConnectSupported"`
	FederationFlags                int    `json:"federationFlags"`
	IsSignup                       bool   `json:"isSignup"`
	FlowToken                      string `json:"flowToken"`
	IsAccessPassSupported          bool   `json:"isAccessPassSupported"`
}

// GetCredentialTypeResponse is the response from credential type detection
type GetCredentialTypeResponse struct {
	Username       string `json:"Username"`
	Display        string `json:"Display"`
	IfExistsResult int    `json:"IfExistsResult"`
	IsUnmanaged    bool   `json:"IsUnmanaged"`
	ThrottleStatus int    `json:"ThrottleStatus"`
	Credentials    struct {
		PrefCredential        int         `json:"PrefCredential"`
		HasPassword           bool        `json:"HasPassword"`
		RemoteNgcParams       interface{} `json:"RemoteNgcParams"`
		FidoParams            interface{} `json:"FidoParams"`
		SasParams             interface{} `json:"SasParams"`
		CertAuthParams        interface{} `json:"CertAuthParams"`
		GoogleParams          interface{} `json:"GoogleParams"`
		FacebookParams        interface{} `json:"FacebookParams"`
		FederationRedirectURL string      `json:"FederationRedirectUrl"`
	} `json:"Credentials"`
	FlowToken          string `json:"FlowToken"`
	IsSignupDisallowed bool   `json:"IsSignupDisallowed"`
	APICanary          string `json:"apiCanary"`
}

// MFARequest is the request body for MFA operations
type MFARequest struct {
	AuthMethodID       string `json:"AuthMethodId"`
	Method             string `json:"Method"`
	Ctx                string `json:"Ctx"`
	FlowToken          string `json:"FlowToken"`
	SessionID          string `json:"SessionId,omitempty"`
	AdditionalAuthData string `json:"AdditionalAuthData,omitempty"`
}

// MFAResponse is the response from MFA operations
type MFAResponse struct {
	Success       bool        `json:"Success"`
	ResultValue   string      `json:"ResultValue"`
	Message       interface{} `json:"Message"`
	AuthMethodID  string      `json:"AuthMethodId"`
	ErrCode       int         `json:"ErrCode"`
	Retry         bool        `json:"Retry"`
	FlowToken     string      `json:"FlowToken"`
	Ctx           string      `json:"Ctx"`
	SessionID     string      `json:"SessionId"`
	CorrelationID string      `json:"CorrelationId"`
	Timestamp     time.Time   `json:"Timestamp"`
	Entropy       int         `json:"Entropy"`
}

// UserProof represents an available MFA method for the user
type UserProof struct {
	AuthMethodID string `json:"authMethodId"`
	Data         string `json:"data"`
	Display      string `json:"display"`
	IsDefault    bool   `json:"isDefault"`
}

// MFA method IDs
const (
	MFAPhoneAppOTP          = "PhoneAppOTP"
	MFAPhoneAppNotification = "PhoneAppNotification"
	MFAOneWaySMS            = "OneWaySMS"
	MFATwoWayVoiceMobile    = "TwoWayVoiceMobile"
)
