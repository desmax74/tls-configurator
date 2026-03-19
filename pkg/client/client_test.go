package client

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestValidateTLSProfile(t *testing.T) {
	tests := []struct {
		name        string
		profile     *configv1.TLSSecurityProfile
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil profile",
			profile:     nil,
			expectError: true,
			errorMsg:    "TLS security profile cannot be nil",
		},
		{
			name: "valid custom profile with TLS 1.3",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
						MinTLSVersion: configv1.VersionTLS13,
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid custom profile with TLS 1.2",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						Ciphers:       []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
						MinTLSVersion: configv1.VersionTLS12,
					},
				},
			},
			expectError: false,
		},
		{
			name: "custom profile without custom config",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
			},
			expectError: true,
			errorMsg:    "custom TLS profile type requires custom configuration",
		},
		{
			name: "custom profile without ciphers",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS13,
					},
				},
			},
			expectError: true,
			errorMsg:    "custom profile must specify at least one cipher",
		},
		{
			name: "custom profile without min TLS version",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						Ciphers: []string{"TLS_AES_128_GCM_SHA256"},
					},
				},
			},
			expectError: true,
			errorMsg:    "custom profile must specify minimum TLS version",
		},
		{
			name: "valid intermediate profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileIntermediateType,
			},
			expectError: false,
		},
		{
			name: "valid modern profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileModernType,
			},
			expectError: false,
		},
		{
			name: "valid old profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileOldType,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSProfile(tt.profile)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateCustomProfile(t *testing.T) {
	tests := []struct {
		name        string
		custom      *configv1.CustomTLSProfile
		expectError bool
	}{
		{
			name:        "nil custom profile",
			custom:      nil,
			expectError: true,
		},
		{
			name: "valid custom profile",
			custom: &configv1.CustomTLSProfile{
				TLSProfileSpec: configv1.TLSProfileSpec{
					Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
					MinTLSVersion: configv1.VersionTLS13,
				},
			},
			expectError: false,
		},
		{
			name: "empty ciphers",
			custom: &configv1.CustomTLSProfile{
				TLSProfileSpec: configv1.TLSProfileSpec{
					Ciphers:       []string{},
					MinTLSVersion: configv1.VersionTLS13,
				},
			},
			expectError: true,
		},
		{
			name: "invalid TLS version",
			custom: &configv1.CustomTLSProfile{
				TLSProfileSpec: configv1.TLSProfileSpec{
					Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
					MinTLSVersion: "VersionTLS14", // Invalid version
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomProfile(tt.custom)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
