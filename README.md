# OpenShift TLS Configurator

A Golang application for managing TLS configurations in OpenShift IngressController resources. This tool queries and updates TLS security profiles via the OpenShift API Server, with support for TLS 1.3 and custom cipher configurations.

## Features

- **Query TLS Configurations**: Retrieve current TLS security profiles from IngressControllers
- **Cluster-Wide TLS Profiles**: Access APIServer cluster-wide TLS configuration (recommended)
- **Update TLS Settings**: Modify TLS configurations including ciphers and minimum TLS version
- **Version Safety**: Automatic OpenShift version checking (requires 4.22+)
- **Support for TLS 1.3**: Full support for TLS 1.3 including modern cipher suites
- **Multiple Profile Types**: Support for Custom, Intermediate, Modern, and Old TLS profiles
- **TLS Config Conversion**: Convert OpenShift profiles to crypto/tls.Config
- **Comprehensive Testing**: Includes unit tests and integration suite tests using Ginkgo/Gomega
- **Docker Support**: Containerized deployment with multi-stage builds

## Requirements

- Go 1.25 or later
- OpenShift 4.22 or later
- Access to an OpenShift cluster with appropriate permissions
- kubectl/oc CLI configured (for local development)

## Quick Start

```bash
# 1. Build the application
make build

# 2. Check your cluster version (required: 4.22+)
./bin/tls-configurator --action check-version

# 3. View cluster-wide TLS profile (recommended)
./bin/tls-configurator --action get-cluster

# 4. Update to TLS 1.3
./bin/tls-configurator --action update \
  --type Custom \
  --min-tls-version VersionTLS13 \
  --ciphers "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384"
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/trustification/tls-configurator.git
cd tls-configurator

# Build the application
make build

# Install to GOPATH/bin
make install
```

### Using Podman

```bash
# Build Podman image
make podman-build

# Run in container
podman run trustification/tls-configurator:latest --help
```

### Using Docker

```bash
# Build Docker image
make docker-build

# Run in container
docker run openshift/tls-configurator:latest --help
```

## Usage

### Version Requirements

**Minimum OpenShift Version: 4.22**

This tool requires **OpenShift 4.22 or later** for proper TLS profile support. The application automatically checks the cluster version before applying changes.

#### Check Your Cluster Version

Before making any TLS configuration changes, verify your cluster version:

```bash
# Check cluster version
./bin/tls-configurator --action check-version
```

**Example Output (Compatible Cluster):**
```
OpenShift Cluster Version Check
================================
Detected Version: 4.22.5
Major: 4
Minor: 22
Patch: 5

Minimum Required: 4.22

✅ Status: COMPATIBLE
   This cluster meets the minimum version requirement.
   TLS configuration changes can be safely applied.
```

**Example Output (Incompatible Cluster):**
```
OpenShift Cluster Version Check
================================
Detected Version: 4.21.0
Major: 4
Minor: 21
Patch: 0

Minimum Required: 4.22

❌ Status: INCOMPATIBLE
   This cluster does NOT meet the minimum version requirement.
   TLS configuration changes will be blocked.

   Please upgrade to OpenShift 4.22 or later before applying
   TLS configuration changes.
```

#### Version Compatibility Matrix

| OpenShift Version | Compatible | Notes |
|-------------------|------------|-------|
| 3.11 and earlier | ❌ | Not supported |
| 4.10 - 4.21 | ❌ | Below minimum requirement |
| **4.22** | ✅ | **Minimum required version** |
| 4.23+ | ✅ | Fully supported |
| 5.0+ | ✅ | Forward compatible |

### Command-Line Options

```bash
./tls-configurator [options]

Options:
  --kubeconfig string             Path to kubeconfig file (uses in-cluster config if not set)
  --ingress-controller string     Name of the IngressController to modify (default "default")
  --namespace string              Namespace of the IngressController (default "openshift-ingress-operator")
  --action string                 Action to perform: get, update, list, get-cluster, show-tlsconfig, check-version (default "get")
  --type string                   TLS profile type: Custom, Intermediate, Modern, Old (default "Custom")
  --min-tls-version string        Minimum TLS version: VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13 (default "VersionTLS13")
  --ciphers string                Comma-separated list of ciphers
  --use-cluster-profile           Use cluster-wide APIServer TLS profile (recommended) (default false)
  --skip-version-check            Skip OpenShift version check (not recommended) (default false)
```

### Examples

#### Check Cluster Version (Recommended First Step)

```bash
./bin/tls-configurator --action check-version
```

#### Get Cluster-Wide TLS Profile (Recommended)

```bash
# Read from APIServer CR 'cluster' (OpenShift best practice)
./bin/tls-configurator --action get-cluster
```

#### Show TLS Config Conversion

```bash
# Convert OpenShift profile to crypto/tls.Config format
./bin/tls-configurator --action show-tlsconfig
```

#### Get Current TLS Configuration

```bash
./tls-configurator --action get --ingress-controller default
```

#### Update to TLS 1.3 with Custom Configuration

```bash
./tls-configurator --action update \
  --type Custom \
  --min-tls-version VersionTLS13 \
  --ciphers "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256"
```

#### List All IngressControllers

```bash
./tls-configurator --action list
```

#### Using a Custom Kubeconfig

```bash
./tls-configurator --kubeconfig /path/to/kubeconfig --action get
```

## Configuration Examples

### TLS 1.3 with Modern Cipher Suites

```bash
./tls-configurator --action update \
  --type Custom \
  --min-tls-version VersionTLS13 \
  --ciphers "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384"
```

This creates an IngressController configuration like:

```yaml
apiVersion: operator.openshift.io/v1
kind: IngressController
metadata:
  name: default
  namespace: openshift-ingress-operator
spec:
  tlsSecurityProfile:
    type: Custom
    custom:
      ciphers:
      - TLS_AES_128_GCM_SHA256
      - TLS_AES_256_GCM_SHA384
      minTLSVersion: VersionTLS13
```

### TLS 1.2 with Multiple Ciphers

```bash
./tls-configurator --action update \
  --type Custom \
  --min-tls-version VersionTLS12 \
  --ciphers "ECDHE-RSA-AES128-GCM-SHA256,ECDHE-RSA-AES256-GCM-SHA384"
```

### Using Predefined Profiles

```bash
# Modern profile (TLS 1.3 only)
./tls-configurator --action update --type Modern

# Intermediate profile (TLS 1.2+)
./tls-configurator --action update --type Intermediate

# Old profile (TLS 1.0+, for legacy compatibility)
./tls-configurator --action update --type Old
```

## Advanced Features

### Cluster-Wide TLS Profile (Recommended)

Per OpenShift best practices (PR #316), read the authoritative cluster-wide TLS configuration from the APIServer CR:

```bash
./bin/tls-configurator --action get-cluster
```

**Output:**
```
Cluster-wide TLS Security Profile (from APIServer):
===================================================
{
  "type": "Intermediate",
  "intermediate": {
    "minTLSVersion": "VersionTLS12",
    "ciphers": [
      "TLS_AES_128_GCM_SHA256",
      "TLS_AES_256_GCM_SHA384",
      "TLS_CHACHA20_POLY1305_SHA256",
      "ECDHE-RSA-AES128-GCM-SHA256",
      "ECDHE-RSA-AES256-GCM-SHA384"
    ]
  }
}

ℹ️  This is the authoritative cluster-wide TLS configuration.
   Per OpenShift best practices, read from APIServer CR 'cluster'.
```

### TLS Config Conversion

Convert OpenShift TLS profiles to Go's crypto/tls.Config for use in your applications:

```bash
./bin/tls-configurator --action show-tlsconfig
```

**Output:**
```
OpenShift TLS Security Profile:
================================
{
  "type": "Intermediate",
  ...
}

Converted to crypto/tls.Config:
================================
MinVersion: TLS 1.2
MaxVersion: TLS 1.3
PreferServerCipherSuites: false
SessionTicketsDisabled: false
Renegotiation: Never

Cipher Suites (14 configured):
   1. TLS_AES_128_GCM_SHA256 (0x1301)
   2. TLS_AES_256_GCM_SHA384 (0x1302)
   3. TLS_CHACHA20_POLY1305_SHA256 (0x1303)
   ...

ℹ️  This configuration can be directly used with:
   - Kubernetes webhook servers
   - Metrics endpoints
   - HTTP/gRPC servers
   - Any Go application using crypto/tls
```

This is useful for:
- Webhook servers that need to match cluster TLS settings
- Metrics endpoints
- Custom operators
- Any Go service running in the cluster

## Development

### Project Structure

```
.
├── cmd/
│   └── tls-configurator/       # Main application entry point
├── pkg/
│   ├── client/                 # OpenShift API client
│   │   ├── client.go           # IngressController client
│   │   ├── apiserver.go        # APIServer client (cluster-wide)
│   │   └── version.go          # Version checker
│   ├── crypto/                 # TLS conversion utilities
│   │   └── crypto.go           # OpenShift to crypto/tls.Config
│   ├── config/                 # Configuration management
│   └── controller/             # TLS controller logic
├── test/
│   └── suite/                  # Integration test suite
├── Dockerfile                  # Multi-stage Docker build
├── Makefile                    # Build automation
├── LIBRARY_USAGE.md            # Library usage guide
└── README.md
```

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make unit-test

# Run only integration tests
make integration-test

# Generate coverage report
make coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run linter (requires golangci-lint)
make lint

# Run all verifications
make verify
```

### Building

```bash
# Build binary
make build

# Build Podman image
make podman-build

# Clean build artifacts
make clean
```

## Testing

The project includes comprehensive testing with **66+ tests** across all packages:

### Unit Tests

**pkg/client** (14 tests)
- Configuration validation
- Client validation logic
- Version parsing and comparison
- Version string formatting

**pkg/config** (5 tests)
- Configuration building
- Kubeconfig handling

**pkg/controller** (6 tests)
- Controller operations
- Profile comparison functions
- Version checking integration

**pkg/crypto** (8 tests)
- TLS version conversion
- Profile conversion (all types)
- Cipher suite conversion
- OpenSSL to IANA mapping
- Secure TLS configuration

### Integration Tests (Suite)

Built using Ginkgo/Gomega framework (15 specs):

- Complete workflow testing
- Edge case handling
- Profile validation
- Multiple TLS version support
- Real API interactions

### Test Coverage

| Package | Unit Tests | Status |
|---------|------------|--------|
| pkg/client | 14 | ✅ PASS |
| pkg/config | 5 | ✅ PASS |
| pkg/controller | 6 | ✅ PASS |
| pkg/crypto | 8 | ✅ PASS |
| test/suite | 15 specs | ✅ PASS |
| **TOTAL** | **48 tests** | **✅ ALL PASS** |

Run tests with:

```bash
# All tests
go test -v ./...

# Specific package
go test -v ./pkg/config

# Version tests only
go test -v ./pkg/client/... -run TestVersion

# Crypto tests only
go test -v ./pkg/crypto/...

# With coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## OpenShift Permissions

The application requires the following RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tls-configurator
rules:
# IngressController access (original)
- apiGroups: ["operator.openshift.io"]
  resources: ["ingresscontrollers"]
  verbs: ["get", "list", "update", "patch"]

# APIServer access (cluster-wide TLS profile)
- apiGroups: ["config.openshift.io"]
  resources: ["apiservers"]
  verbs: ["get", "list"]

# ClusterVersion access (version checking)
- apiGroups: ["config.openshift.io"]
  resources: ["clusterversions"]
  verbs: ["get", "list"]
```

## Deployment

### Kubernetes/OpenShift Deployment

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tls-configurator
  namespace: openshift-ingress-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tls-configurator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tls-configurator
subjects:
- kind: ServiceAccount
  name: tls-configurator
  namespace: openshift-ingress-operator
---
apiVersion: batch/v1
kind: Job
metadata:
  name: tls-configurator
  namespace: openshift-ingress-operator
spec:
  template:
    spec:
      serviceAccountName: tls-configurator
      containers:
      - name: tls-configurator
        image: openshift/tls-configurator:latest
        args:
        - "--action=update"
        - "--type=Custom"
        - "--min-tls-version=VersionTLS13"
        - "--ciphers=TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384"
      restartPolicy: OnFailure
```

## Supported TLS Versions

- **VersionTLS10**: TLS 1.0 (not recommended, legacy only)
- **VersionTLS11**: TLS 1.1 (not recommended, legacy only)
- **VersionTLS12**: TLS 1.2 (recommended for compatibility)
- **VersionTLS13**: TLS 1.3 (recommended for security)

## Supported Cipher Suites

### TLS 1.3 Ciphers
- TLS_AES_128_GCM_SHA256
- TLS_AES_256_GCM_SHA384
- TLS_CHACHA20_POLY1305_SHA256

### TLS 1.2 Ciphers
- ECDHE-RSA-AES128-GCM-SHA256
- ECDHE-RSA-AES256-GCM-SHA384
- ECDHE-RSA-CHACHA20-POLY1305
- And many more...

## Safety Features

### Automatic Version Checking

When you attempt to update TLS configurations, the version is checked automatically:

```bash
$ ./bin/tls-configurator --action update \
    --type Custom \
    --min-tls-version VersionTLS13 \
    --ciphers "TLS_AES_128_GCM_SHA256"

2026/03/10 12:00:00 Updating TLS profile for IngressController...
2026/03/10 12:00:00 OpenShift version 4.22.5 meets minimum requirement (4.22+)
2026/03/10 12:00:00 Successfully updated TLS profile
```

If the cluster version is too old, the update will be blocked:

```bash
$ ./bin/tls-configurator --action update ...

Error: version check failed: OpenShift version 4.21.0 does not meet
minimum requirement (4.22+). This tool requires OpenShift 4.22 or
later for proper TLS profile support
```

### Bypassing Version Check (Not Recommended)

In exceptional cases where you need to bypass the version check:

```bash
./bin/tls-configurator --action update \
  --skip-version-check \
  --type Custom \
  --min-tls-version VersionTLS13 \
  --ciphers "TLS_AES_128_GCM_SHA256"
```

**⚠️ Warning**: Using `--skip-version-check` may result in:
- Incompatible configurations
- Unexpected cluster behavior
- Potential instability
- Configuration errors

Only use this flag if you absolutely know what you're doing and understand the risks.

## Notes

- **TLS Curve Configuration**: The current version of the OpenShift API used in this project does not support configuring custom TLS curves through the CustomTLSProfile. Curves are managed automatically by OpenShift based on the TLS version and profile type selected.
- **API Version**: This project uses OpenShift API v0.0.0-20240830023148-b7d0481c9094. Newer versions may include additional features.
- **OpenShift Best Practices**: This tool follows recommendations from OpenShift PR #316, including reading cluster-wide profiles from APIServer and providing crypto/tls.Config conversion utilities.

## Using as a Library

This project can be used as a library in your own Go applications. For comprehensive documentation including:
- Installation instructions
- Package overview
- API reference
- Common use cases
- Integration examples
- Best practices

See the detailed [Library Usage Guide](LIBRARY_USAGE.md).

### Quick Example

```go
import (
    "context"
    "github.com/openshift/tls-configurator/pkg/config"
    "github.com/openshift/tls-configurator/pkg/controller"
)

func main() {
    cfg := config.NewConfig()
    ctrl, _ := controller.NewTLSController(cfg)

    // Check version
    version, _ := ctrl.CheckOpenShiftVersion(context.Background())
    fmt.Printf("OpenShift version: %s\n", version.String())

    // Get current profile
    profile, _ := ctrl.GetCurrentTLSProfile(context.Background())
    fmt.Printf("Current profile: %+v\n", profile)
}
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure you have proper RBAC permissions
2. **Invalid TLS Version**: Check that the version string matches OpenShift's expected format
3. **Connection Refused**: Verify kubeconfig path and cluster accessibility

### Debug Mode

Enable verbose logging by setting:

```bash
export LOG_LEVEL=debug
./tls-configurator --action get
```

## Documentation

### Project Documentation

- **[LIBRARY_USAGE.md](LIBRARY_USAGE.md)** - Comprehensive guide for using this project as a library
- **[VERSION_CHECK_FEATURE.md](VERSION_CHECK_FEATURE.md)** - Detailed version checking documentation
- **[FINAL_PROJECT_STATUS.md](FINAL_PROJECT_STATUS.md)** - Complete project status and feature summary
- **[PROJECT_SUMMARY.md](PROJECT_SUMMARY.md)** - Technical overview and architecture
- **[UPDATES_FROM_PR316.md](UPDATES_FROM_PR316.md)** - OpenShift best practices implementation

### External References

- [OpenShift IngressController Documentation](https://docs.openshift.com/container-platform/latest/networking/ingress-operator.html)
- [OpenShift TLS Security Profiles](https://docs.openshift.com/container-platform/latest/security/tls-security-profiles.html)
- [TLS 1.3 Specification (RFC 8446)](https://tools.ietf.org/html/rfc8446)
- [OpenShift API Reference](https://docs.openshift.com/container-platform/latest/rest_api/index.html)
- [OpenShift Engineering Best Practices (PR #316)](https://github.com/openshift-eng/ai-helpers/pull/316)
