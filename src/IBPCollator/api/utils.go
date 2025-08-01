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
	// Sanitize and validate countries
	for i, country := range filter.Countries {
		filter.Countries[i] = sanitizeString(country)
		if !validateCountryCode(filter.Countries[i]) {
			return fmt.Errorf("invalid country code: %s", filter.Countries[i])
		}
	}

	// Sanitize and validate ASNs
	for i, asn := range filter.ASNs {
		filter.ASNs[i] = sanitizeString(asn)
		if !validateASN(filter.ASNs[i]) {
			return fmt.Errorf("invalid ASN: %s", filter.ASNs[i])
		}
	}

	// Sanitize and validate networks
	for i, network := range filter.Networks {
		filter.Networks[i] = sanitizeString(network)
		// Networks can be partial matches, so more lenient validation
		if len(filter.Networks[i]) > 100 {
			return fmt.Errorf("network name too long")
		}
	}

	// Sanitize and validate services
	for i, service := range filter.Services {
		filter.Services[i] = sanitizeString(service)
		if !validateIdentifier(filter.Services[i]) {
			return fmt.Errorf("invalid service name: %s", filter.Services[i])
		}
	}

	// Sanitize and validate members
	for i, member := range filter.Members {
		filter.Members[i] = sanitizeString(member)
		if !validateMemberName(filter.Members[i]) {
			return fmt.Errorf("invalid member name: %s", filter.Members[i])
		}
	}

	// Sanitize and validate domains
	for i, domain := range filter.Domains {
		filter.Domains[i] = sanitizeString(domain)
		// Domain can contain dots, so we use a more permissive check
		if !safeIdentifierRegex.MatchString(filter.Domains[i]) {
			return fmt.Errorf("invalid domain name: %s", filter.Domains[i])
		}
	}

	// Limit total number of filters to prevent abuse
	totalFilters := len(filter.Countries) + len(filter.ASNs) + len(filter.Networks) +
		len(filter.Services) + len(filter.Members) + len(filter.Domains)
	if totalFilters > 50 {
		return fmt.Errorf("too many filters specified (max 50 total)")
	}

	return nil
}
