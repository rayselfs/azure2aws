package provider

// Provider interface defines the contract for SAML identity providers
type Provider interface {
	// Authenticate performs authentication and returns the SAML assertion
	Authenticate(creds *LoginCredentials) (string, error)
}

// LoginCredentials contains the credentials for authentication
type LoginCredentials struct {
	Username string
	Password string
	MFAToken string // Optional MFA token for OTP-based authentication
}

// NewLoginCredentials creates a new LoginCredentials instance
func NewLoginCredentials(username, password string) *LoginCredentials {
	return &LoginCredentials{
		Username: username,
		Password: password,
	}
}
