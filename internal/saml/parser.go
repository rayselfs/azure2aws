package saml

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

const (
	// AWS role attribute name in SAML assertions
	awsRoleAttributeName = "https://aws.amazon.com/SAML/Attributes/Role"
	// AWS session duration attribute name
	awsSessionDurationAttributeName = "https://aws.amazon.com/SAML/Attributes/SessionDuration"
)

// ExtractRoles extracts AWS roles from a base64-encoded SAML assertion
func ExtractRoles(samlAssertion string) ([]string, error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAML assertion: %w", err)
	}

	return extractRolesFromXML(decoded)
}

// extractRolesFromXML extracts AWS roles from SAML XML
func extractRolesFromXML(xmlData []byte) ([]string, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, fmt.Errorf("failed to parse SAML XML: %w", err)
	}

	roles := make([]string, 0)

	// Find Attribute elements with the AWS role name
	for _, attr := range doc.FindElements("//Attribute") {
		name := attr.SelectAttrValue("Name", "")
		if name != awsRoleAttributeName {
			continue
		}

		// Extract AttributeValue elements
		for _, attrValue := range attr.SelectElements("AttributeValue") {
			roleText := strings.TrimSpace(attrValue.Text())
			if roleText != "" {
				roles = append(roles, roleText)
			}
		}
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf("no AWS roles found in SAML assertion")
	}

	return roles, nil
}

// ExtractSessionDuration extracts the session duration from a SAML assertion
// Returns 0 if not found
func ExtractSessionDuration(samlAssertion string) (int64, error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return 0, fmt.Errorf("failed to decode SAML assertion: %w", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(decoded); err != nil {
		return 0, fmt.Errorf("failed to parse SAML XML: %w", err)
	}

	// Find the session duration attribute
	for _, attr := range doc.FindElements("//Attribute") {
		name := attr.SelectAttrValue("Name", "")
		if name != awsSessionDurationAttributeName {
			continue
		}

		// Get the first AttributeValue
		attrValue := attr.SelectElement("AttributeValue")
		if attrValue != nil {
			var duration int64
			text := strings.TrimSpace(attrValue.Text())
			if _, err := fmt.Sscanf(text, "%d", &duration); err == nil {
				return duration, nil
			}
		}
	}

	return 0, nil // Not found, return 0 (will use default)
}

// ExtractDestination extracts the destination URL from a SAML assertion
func ExtractDestination(samlAssertion string) (string, error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return "", fmt.Errorf("failed to decode SAML assertion: %w", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(decoded); err != nil {
		return "", fmt.Errorf("failed to parse SAML XML: %w", err)
	}

	// Find Response element and get Destination attribute
	response := doc.SelectElement("Response")
	if response != nil {
		dest := response.SelectAttrValue("Destination", "")
		if dest != "" {
			return dest, nil
		}
	}

	// Try samlp:Response
	response = doc.FindElement("//Response")
	if response != nil {
		dest := response.SelectAttrValue("Destination", "")
		if dest != "" {
			return dest, nil
		}
	}

	return "", nil
}

// ParseAssertion is a convenience function that extracts and parses roles from a SAML assertion
func ParseAssertion(samlAssertion string) ([]*AWSRole, error) {
	roleStrings, err := ExtractRoles(samlAssertion)
	if err != nil {
		return nil, err
	}

	return ParseAWSRoles(roleStrings)
}
