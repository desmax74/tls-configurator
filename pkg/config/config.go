package config

import (
	"fmt"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents the application configuration
type Config struct {
	Kubeconfig            string
	IngressControllerName string
	Namespace             string
	TLSProfile            *configv1.TLSSecurityProfile
}

// TLSConfig represents the desired TLS configuration
type TLSConfig struct {
	Type          configv1.TLSProfileType
	Ciphers       []string
	MinTLSVersion configv1.TLSProtocolVersion
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Kubeconfig:            os.Getenv("KUBECONFIG"),
		IngressControllerName: "default",
		Namespace:             "openshift-ingress-operator",
	}
}

// GetKubeConfig returns a Kubernetes REST config (method)
func (c *Config) GetKubeConfig() (*rest.Config, error) {
	return GetKubeConfig(c.Kubeconfig)
}

// GetKubeConfig returns a Kubernetes REST config from kubeconfig path (standalone function)
func GetKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
		}
	} else {
		// Try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
		}
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.IngressControllerName == "" {
		return fmt.Errorf("ingress controller name cannot be empty")
	}
	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	return nil
}

// BuildTLSProfile creates a TLS security profile from TLSConfig
func BuildTLSProfile(tlsConfig *TLSConfig) *configv1.TLSSecurityProfile {
	if tlsConfig == nil {
		return nil
	}

	profile := &configv1.TLSSecurityProfile{
		Type: tlsConfig.Type,
	}

	if tlsConfig.Type == configv1.TLSProfileCustomType {
		customProfile := &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				Ciphers:       tlsConfig.Ciphers,
				MinTLSVersion: tlsConfig.MinTLSVersion,
			},
		}

		profile.Custom = customProfile
	}

	return profile
}

// GetScheme returns a runtime scheme with OpenShift API types registered
func GetScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := operatorv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add operator scheme: %w", err)
	}
	if err := configv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add config scheme: %w", err)
	}
	return scheme, nil
}
