package crypto

import (
	"crypto/tls"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
)

// ConvertTLSProfile converts an OpenShift TLS security profile to a crypto/tls.Config
func ConvertTLSProfile(profile *configv1.TLSSecurityProfile) (*tls.Config, error) {
	if profile == nil {
		// Use default intermediate profile
		return GetDefaultTLSConfig(), nil
	}

	var minVersion uint16
	var cipherSuites []uint16
	var err error

	switch profile.Type {
	case configv1.TLSProfileCustomType:
		if profile.Custom == nil {
			return nil, fmt.Errorf("custom TLS profile requires custom configuration")
		}
		minVersion, cipherSuites, err = convertCustomProfile(profile.Custom)
		if err != nil {
			return nil, err
		}

	case configv1.TLSProfileModernType:
		minVersion = tls.VersionTLS13
		cipherSuites = GetModernCipherSuites()

	case configv1.TLSProfileIntermediateType:
		minVersion = tls.VersionTLS12
		cipherSuites = GetIntermediateCipherSuites()

	case configv1.TLSProfileOldType:
		minVersion = tls.VersionTLS10
		cipherSuites = GetOldCipherSuites()

	default:
		return nil, fmt.Errorf("unknown TLS profile type: %s", profile.Type)
	}

	tlsConfig := &tls.Config{
		MinVersion:   minVersion,
		CipherSuites: cipherSuites,
	}

	// Apply secure baseline configuration
	SecureTLSConfig(tlsConfig)

	return tlsConfig, nil
}

// convertCustomProfile converts a custom TLS profile to TLS config values
func convertCustomProfile(custom *configv1.CustomTLSProfile) (uint16, []uint16, error) {
	if custom == nil {
		return 0, nil, fmt.Errorf("custom profile cannot be nil")
	}

	// Convert TLS version
	minVersion, err := TLSVersion(string(custom.MinTLSVersion))
	if err != nil {
		return 0, nil, fmt.Errorf("invalid TLS version: %w", err)
	}

	// Convert cipher suites
	ianaNames := OpenSSLToIANACipherSuites(custom.Ciphers)
	cipherSuites, err := CipherSuites(ianaNames)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid cipher suites: %w", err)
	}

	return minVersion, cipherSuites, nil
}

// TLSVersion converts a TLS version string to the corresponding crypto/tls constant
// This implements the pattern recommended by OpenShift library-go
func TLSVersion(version string) (uint16, error) {
	switch configv1.TLSProtocolVersion(version) {
	case configv1.VersionTLS10:
		return tls.VersionTLS10, nil
	case configv1.VersionTLS11:
		return tls.VersionTLS11, nil
	case configv1.VersionTLS12:
		return tls.VersionTLS12, nil
	case configv1.VersionTLS13:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unknown TLS version: %s", version)
	}
}

// OpenSSLToIANACipherSuites converts OpenSSL cipher names to IANA names
// This implements the pattern from OpenShift library-go/pkg/crypto
func OpenSSLToIANACipherSuites(opensslNames []string) []string {
	ianaNames := make([]string, 0, len(opensslNames))

	for _, opensslName := range opensslNames {
		ianaName := opensslToIANAMapping(opensslName)
		if ianaName != "" {
			ianaNames = append(ianaNames, ianaName)
		} else {
			// If no mapping found, assume it's already an IANA name
			ianaNames = append(ianaNames, opensslName)
		}
	}

	return ianaNames
}

// CipherSuites converts IANA cipher suite names to crypto/tls constants
func CipherSuites(ianaNames []string) ([]uint16, error) {
	return convertCipherSuitesFallback(ianaNames)
}

// CipherSuitesOrDie is like CipherSuites but panics on error
func CipherSuitesOrDie(ianaNames []string) []uint16 {
	suites, err := CipherSuites(ianaNames)
	if err != nil {
		panic(fmt.Sprintf("failed to convert cipher suites: %v", err))
	}
	return suites
}

// SecureTLSConfig applies secure baseline settings to a TLS config
func SecureTLSConfig(config *tls.Config) {
	if config == nil {
		return
	}

	// Set secure defaults
	config.PreferServerCipherSuites = true
	config.SessionTicketsDisabled = false
	config.Renegotiation = tls.RenegotiateNever

	// Note: We don't override MinVersion here to allow Old profile
	// to use TLS 1.0/1.1 if explicitly configured
}

// GetDefaultTLSConfig returns the default (Intermediate) TLS configuration
func GetDefaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: GetIntermediateCipherSuites(),
	}
}

// DefaultTLSVersion returns the default TLS version (1.2)
func DefaultTLSVersion() uint16 {
	return tls.VersionTLS12
}

// GetModernCipherSuites returns cipher suites for the Modern profile (TLS 1.3)
func GetModernCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}
}

// GetIntermediateCipherSuites returns cipher suites for the Intermediate profile
func GetIntermediateCipherSuites() []uint16 {
	return []uint16{
		// TLS 1.3 ciphers
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,

		// TLS 1.2 ciphers
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
}

// GetOldCipherSuites returns cipher suites for the Old profile
func GetOldCipherSuites() []uint16 {
	suites := GetIntermediateCipherSuites()
	// Add additional legacy ciphers
	suites = append(suites,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	)
	return suites
}

// opensslToIANAMapping converts OpenSSL cipher names to IANA names
func opensslToIANAMapping(opensslName string) string {
	// Common OpenSSL to IANA cipher suite name mappings
	mapping := map[string]string{
		// TLS 1.3 ciphers (same in both)
		"TLS_AES_128_GCM_SHA256":       "TLS_AES_128_GCM_SHA256",
		"TLS_AES_256_GCM_SHA384":       "TLS_AES_256_GCM_SHA384",
		"TLS_CHACHA20_POLY1305_SHA256": "TLS_CHACHA20_POLY1305_SHA256",

		// OpenSSL format to IANA format for TLS 1.2
		"ECDHE-ECDSA-AES128-GCM-SHA256": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"ECDHE-RSA-AES128-GCM-SHA256":   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"ECDHE-ECDSA-AES256-GCM-SHA384": "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"ECDHE-RSA-AES256-GCM-SHA384":   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"ECDHE-ECDSA-CHACHA20-POLY1305": "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
		"ECDHE-RSA-CHACHA20-POLY1305":   "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
		"ECDHE-ECDSA-AES128-SHA256":     "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		"ECDHE-RSA-AES128-SHA256":       "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		"ECDHE-ECDSA-AES128-SHA":        "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		"ECDHE-RSA-AES128-SHA":          "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		"ECDHE-ECDSA-AES256-SHA":        "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		"ECDHE-RSA-AES256-SHA":          "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"AES128-GCM-SHA256":             "TLS_RSA_WITH_AES_128_GCM_SHA256",
		"AES256-GCM-SHA384":             "TLS_RSA_WITH_AES_256_GCM_SHA384",
		"AES128-SHA256":                 "TLS_RSA_WITH_AES_128_CBC_SHA256",
		"AES128-SHA":                    "TLS_RSA_WITH_AES_128_CBC_SHA",
		"AES256-SHA":                    "TLS_RSA_WITH_AES_256_CBC_SHA",
	}

	// Try exact match
	if ianaName, ok := mapping[opensslName]; ok {
		return ianaName
	}

	// Try case-insensitive match
	for openssl, iana := range mapping {
		if strings.EqualFold(opensslName, openssl) {
			return iana
		}
	}

	// Return empty string if no mapping found
	return ""
}

// convertCipherSuitesFallback is a fallback cipher suite converter
func convertCipherSuitesFallback(ianaNames []string) ([]uint16, error) {
	mapping := map[string]uint16{
		// TLS 1.3
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

		// TLS 1.2 ECDHE
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,

		// Legacy ciphers
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}

	suites := make([]uint16, 0, len(ianaNames))
	for _, name := range ianaNames {
		if id, ok := mapping[name]; ok {
			suites = append(suites, id)
		} else {
			return nil, fmt.Errorf("unknown cipher suite: %s", name)
		}
	}

	return suites, nil
}
