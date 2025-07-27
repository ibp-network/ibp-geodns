package api

import (
	"fmt"
	"regexp"
	"strings"
)

// Validate and sanitize common inputs
var (
	// Only allow alphanumeric, dash, underscore, and dot for most identifiers
	safeIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	// Member names can have spaces
	safeMemberNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s]+$`)
	// Date format validation
	dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	// Country code validation (2 letter codes)
	countryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)
	// ASN validation
	asnRegex = regexp.MustCompile(`^AS\d+$`)
)

// sanitizeString removes any potentially dangerous characters
func sanitizeString(input string) string {
	// Remove any null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	// Trim whitespace
	input = strings.TrimSpace(input)
	return input
}

// validateIdentifier checks if an identifier is safe to use
func validateIdentifier(input string) bool {
	if input == "" {
		return true // Empty is valid (for optional parameters)
	}
	return safeIdentifierRegex.MatchString(input)
}

// validateMemberName checks if a member name is safe to use (allows spaces)
func validateMemberName(input string) bool {
	if input == "" {
		return true // Empty is valid (for optional parameters)
	}
	return safeMemberNameRegex.MatchString(input)
}

// validateDate checks if a date string is in the correct format
func validateDate(input string) bool {
	if input == "" {
		return true // Empty is valid (for optional parameters)
	}
	return dateRegex.MatchString(input)
}

// validateCountryCode checks if a country code is valid
func validateCountryCode(input string) bool {
	if input == "" {
		return true
	}
	return countryCodeRegex.MatchString(strings.ToUpper(input))
}

// validateASN checks if an ASN is valid
func validateASN(input string) bool {
	if input == "" {
		return true
	}
	return asnRegex.MatchString(input)
}

// sanitizeRequestFilter validates and sanitizes request filters
func sanitizeRequestFilter(filter *RequestFilter) error {
	filter.Country = sanitizeString(filter.Country)
	filter.ASN = sanitizeString(filter.ASN)
	filter.Network = sanitizeString(filter.Network)
	filter.Service = sanitizeString(filter.Service)
	filter.Member = sanitizeString(filter.Member)
	filter.Domain = sanitizeString(filter.Domain)

	if !validateCountryCode(filter.Country) {
		return fmt.Errorf("invalid country code")
	}

	if !validateASN(filter.ASN) {
		return fmt.Errorf("invalid ASN")
	}

	if !validateMemberName(filter.Member) {
		return fmt.Errorf("invalid member name")
	}

	if !validateIdentifier(filter.Service) {
		return fmt.Errorf("invalid service name")
	}

	// Domain can contain dots, so we use a more permissive check
	if filter.Domain != "" && !safeIdentifierRegex.MatchString(filter.Domain) {
		return fmt.Errorf("invalid domain name")
	}

	return nil
}
