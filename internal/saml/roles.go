package saml

import (
	"fmt"
	"strings"
)

// AWSRole represents an AWS IAM role that can be assumed via SAML
type AWSRole struct {
	RoleARN      string // The ARN of the IAM role
	PrincipalARN string // The ARN of the SAML provider
	Name         string // Friendly name extracted from the ARN
}

// ParseAWSRoles parses role strings in the format "PrincipalARN,RoleARN" or "RoleARN,PrincipalARN"
func ParseAWSRoles(roleStrings []string) ([]*AWSRole, error) {
	roles := make([]*AWSRole, 0, len(roleStrings))

	for _, roleStr := range roleStrings {
		role, err := parseRoleString(roleStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse role %q: %w", roleStr, err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// parseRoleString parses a single role string
func parseRoleString(roleStr string) (*AWSRole, error) {
	parts := strings.Split(roleStr, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid role format: expected 2 parts, got %d", len(parts))
	}

	part1 := strings.TrimSpace(parts[0])
	part2 := strings.TrimSpace(parts[1])

	var roleARN, principalARN string

	// Determine which part is the role and which is the principal
	// Role ARNs contain ":role/" and Principal ARNs contain ":saml-provider/"
	if strings.Contains(part1, ":role/") && strings.Contains(part2, ":saml-provider/") {
		roleARN = part1
		principalARN = part2
	} else if strings.Contains(part2, ":role/") && strings.Contains(part1, ":saml-provider/") {
		roleARN = part2
		principalARN = part1
	} else {
		return nil, fmt.Errorf("invalid role/principal ARNs in: %s", roleStr)
	}

	return &AWSRole{
		RoleARN:      roleARN,
		PrincipalARN: principalARN,
		Name:         extractRoleName(roleARN),
	}, nil
}

// extractRoleName extracts the role name from an ARN
// e.g., "arn:aws:iam::123456789012:role/MyRole" -> "MyRole"
func extractRoleName(roleARN string) string {
	parts := strings.Split(roleARN, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return roleARN
}

// String returns a string representation of the role
func (r *AWSRole) String() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.RoleARN)
}

// AccountID extracts the AWS account ID from the role ARN
func (r *AWSRole) AccountID() string {
	// ARN format: arn:aws:iam::ACCOUNT_ID:role/RoleName
	parts := strings.Split(r.RoleARN, ":")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}
