# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**OpenShift TLS Configurator** is a Go CLI that queries and updates TLS security
profiles on OpenShift `IngressController` resources (and reads the cluster-wide
`APIServer` TLS profile). It is packaged as a container image and is consumed by
the **trusted-profile-analyzer-operator** as a pre-install/pre-upgrade Job that
hardens the cluster ingress to TLS 1.3 before RHTPA is deployed.

**Key Facts:**
- Written in Go 1.25.7 (see `go.mod`)
- Depends on `github.com/openshift/api v0.0.0-20240830023148-b7d0481c9094`
- Builds a single static binary (`CGO_ENABLED=0`), shipped on `ubi9/ubi-minimal`
- Published image (dev): `quay.io/mdessi/tls-configurator:latest`
- Requires OpenShift 4.22+ (enforced by a built-in version check)

## Architecture

```
cmd/tls-configurator/   Entry point, flag parsing, action dispatch
pkg/client/             OpenShift/Kubernetes API clients
  client.go             IngressController client
  apiserver.go          Cluster-wide APIServer TLS profile client (+ Watch)
  version.go            ClusterVersion checker (4.22+ gate)
  workloads.go          Deployment rollout client (TLS config-hash annotation)
pkg/config/             Config building / kubeconfig handling
pkg/controller/         One-shot TLS controller orchestration
pkg/crypto/             OpenShift TLSSecurityProfile -> crypto/tls.Config (+ PQC)
pkg/reconcile/          Long-running runtime reconciler (watch -> roll workloads)
test/suite/             Ginkgo/Gomega integration suite
```

Actions (`--action`):
- `get`, `update`, `list`, `get-cluster`, `show-tlsconfig`, `check-version` —
  one-shot operations. `update` writes a `tlsSecurityProfile` onto the
  IngressController; `show-tlsconfig` converts an OpenShift profile into a Go
  `crypto/tls.Config` (see `pkg/crypto/crypto.go`).
- `validate` — reports whether the effective cluster TLS config is
  post-quantum + TLS 1.3 compliant; exits non-zero if not.
- `reconcile` — **long-running** mode: watches the cluster TLS profile and rolls
  target workloads at runtime (see `pkg/reconcile`).

## Development Commands

```bash
make build            # build ./bin/tls-configurator
make test             # all tests
make unit-test        # unit tests only
make integration-test # Ginkgo suite
make fmt vet lint     # code quality
make podman-build     # container image
```

## Post-Quantum Cryptography (PQC) — New Feature

### What PQC means here

Post-quantum security in TLS 1.3 is provided by the **key-exchange (key
agreement) group**, not by the symmetric cipher suite. The target is the hybrid
group **`X25519MLKEM768`** (classical X25519 ECDH combined with ML-KEM-768,
NIST FIPS 203; formerly `X25519Kyber768Draft00`). The AEAD cipher suites
(`TLS_AES_128_GCM_SHA256`, `TLS_AES_256_GCM_SHA384`, …) are unchanged — they are
already quantum-resistant at the symmetric level. **Hybrid PQC key exchange
requires TLS 1.3.**

### What is implemented

- **PQC crypto (`pkg/crypto/crypto.go`).** `PQCCurvePreferences()` returns
  `[X25519MLKEM768, X25519]`. `EnablePQC(*tls.Config)` forces `MinVersion` to
  TLS 1.3 and sets those `CurvePreferences`. `ConvertTLSProfileWithPQC(profile,
  enablePQC)` is the PQC-aware converter (`ConvertTLSProfile` still exists,
  PQC-off, for compatibility). `IsPQCCompliant(*tls.Config)` reports compliance
  with reasons. TLS 1.3 AEAD cipher suites are intentionally left as-is (Go
  ignores `Config.CipherSuites` for TLS 1.3).
- **CLI flags.** `--enable-pqc`, plus reconcile-mode flags `--target-namespace`,
  `--target-deployments`, `--resync-period`. `update` forces `VersionTLS13` when
  `--enable-pqc` is set; `show-tlsconfig` prints the key-exchange groups.
- **`validate` action.** Fails non-zero if the effective cluster profile is not
  PQC/TLS 1.3 compliant — usable in CI and in probes.

### Still requires a follow-up

- **Router-level PQC group is not writable via the pinned API.** The
  `TLSSecurityProfile` type in `openshift/api v0.0.0-20240830...` exposes only
  `minTLSVersion` and `ciphers`, so `update` cannot push a key-exchange group
  onto the IngressController CR. PQC therefore applies to Go services'
  `crypto/tls.Config` (negotiated by default under Go >= 1.24) and to the
  rollout hash — not to the router profile fields. Bump `openshift/api` to a
  release that exposes groups to close this.
- **The image must be rebuilt/republished** (`make podman-build` →
  `quay.io/mdessi/tls-configurator:latest`); the operator pins it by tag.

## Runtime reconciliation (`--action=reconcile`)

`pkg/reconcile` turns the tool from a one-shot CLI into a controller so a TLS
change at runtime updates the operator's workloads:

1. **Source of truth:** the cluster `APIServer` CR (`cluster`)
   `.spec.tlsSecurityProfile`, read via `APIServerClient.GetEffectiveTLSProfile`
   and watched via `APIServerClient.WatchAPIServer`.
2. **Reconcile loop (`Reconciler.Run`):** initial reconcile on startup (covers
   the old pre-install behaviour), then reconcile on every watch event plus a
   periodic `--resync-period` resync; the watch is re-established when the server
   closes it.
3. **Rollout trigger:** `TLSConfigHash(profile, pqc)` (sha256 over the profile +
   PQC flag) is compared to each target Deployment's
   `rhtpa.io/tls-config-hash` pod-template annotation; only Deployments whose
   hash differs are patched (`WorkloadsClient.SetDeploymentTLSHash`), which
   changes the pod template and triggers a rolling restart. This mirrors the
   chart's existing `configHash/auth` pattern. Kubernetes does **not** restart
   pods on ConfigMap/Secret change on its own, hence the explicit hash bump.

**Why an annotation the chart does not manage:** `rhtpa.io/tls-config-hash` is
deliberately absent from the Helm templates so the operator's periodic re-render
does not fight the reconciler (Helm 3-way merge preserves annotations it never
set).

### RBAC

Reconcile mode needs, beyond the one-shot permissions: `watch` on
`config.openshift.io/apiservers`, and `get,list,watch,patch,update` on
`apps/deployments` in the target (release) namespace. The operator chart wires
this up (ClusterRole + namespaced Role).

## Notes for Claude

- Read `pkg/crypto/crypto.go` before touching TLS conversion logic — cipher
  mappings are duplicated in `opensslToIANAMapping` and
  `convertCipherSuitesFallback` and must stay in sync.
- The version gate (`pkg/client/version.go`) blocks `update` below OpenShift
  4.22; use `--skip-version-check` only in tests.
- This tool is deployed by the operator's Helm chart as a long-running
  reconciler Deployment at
  `helm-charts/redhat-trusted-profile-analyzer/templates/init/tls-configure/020-Deployment.yaml`
  (it replaced the old one-shot hook Job). Any new CLI flag must be added there
  and to the RHTPA operator CLAUDE.md.
- Local builds may hit a broken/tampered Go toolchain under `$HOME/go` (a bogus
  `import "uuid"` in `database/sql/driver` and duplicate decls in
  `internal/strconv`). That is a toolchain problem, not this repo's — use a clean
  Go >= 1.24 install to compile.
