package seoregistry

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// canonicalAbsoluteURL rejects ambiguous, credential-bearing, fragment, and
// non-HTTP(S) URLs. Callers compare the returned value to input where exact
// canonical form is required.
func canonicalAbsoluteURL(value string, allowFragment bool) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "\\") || !utf8.ValidString(value) {
		return "", ErrOutputInvalid
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", ErrOutputInvalid
		}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Host == "" ||
		strings.HasSuffix(parsed.Host, ":") || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		(!allowFragment && parsed.Fragment != "") {
		return "", ErrOutputInvalid
	}
	hostname := parsed.Hostname()
	if hostname == "" || hostname != strings.ToLower(hostname) {
		return "", ErrOutputInvalid
	}
	if net.ParseIP(hostname) == nil && !validDNSHostname(hostname) {
		return "", ErrOutputInvalid
	}
	port := parsed.Port()
	if port != "" {
		if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
			return "", ErrOutputInvalid
		}
		number, parseErr := strconv.Atoi(port)
		if parseErr != nil || number < 1 || number > 65535 {
			return "", ErrOutputInvalid
		}
	}
	if parsed.ForceQuery {
		return "", ErrOutputInvalid
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}

func validDNSHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || !dnsLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

func validCanonicalLocale(value string) bool {
	if value == "x-default" {
		return true
	}
	if strings.TrimSpace(value) != value || !localePattern.MatchString(value) {
		return false
	}
	parts := strings.Split(value, "-")
	if parts[0] != strings.ToLower(parts[0]) {
		return false
	}
	for index := 1; index < len(parts); index++ {
		part := parts[index]
		switch {
		case len(part) == 4:
			if part != strings.ToUpper(part[:1])+strings.ToLower(part[1:]) {
				return false
			}
		case len(part) == 2 || len(part) == 3:
			if part != strings.ToUpper(part) {
				return false
			}
		default:
			if part != strings.ToLower(part) {
				return false
			}
		}
	}
	return true
}

func validCanonicalDate(value string) bool {
	if parsed, err := time.Parse(time.DateOnly, value); err == nil && parsed.Format(time.DateOnly) == value {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Format(time.RFC3339) == value
}
