# Container Image Verification Guide

This document describes how to verify the authenticity and integrity of Cartographus container images.

## Prerequisites

Install Cosign:

```bash
# macOS
brew install cosign

# Linux
wget https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign

# Windows
# Download from https://github.com/sigstore/cosign/releases
```

## Verification Process

### Verify Image Signature

All images published to `ghcr.io/tomtom215/cartographus` are signed using keyless signing with GitHub OIDC.

```bash
cosign verify \
  --certificate-identity-regexp='https://github.com/tomtom215/cartographus' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  ghcr.io/tomtom215/cartographus:latest
```

Successful output confirms the image was built from this repository's GitHub Actions workflow and has not been tampered with.

### Verify Specific Version

```bash
cosign verify \
  --certificate-identity-regexp='https://github.com/tomtom215/cartographus' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  ghcr.io/tomtom215/cartographus:v1.0.0
```

### Verify Attestations

Inspect build provenance:

```bash
cosign verify-attestation \
  --certificate-identity-regexp='https://github.com/tomtom215/cartographus' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  --type slsaprovenance \
  ghcr.io/tomtom215/cartographus:latest
```

## Understanding Verification Results

### Successful Verification

```
Verification for ghcr.io/tomtom215/cartographus:latest --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The code-signing certificate was verified using trusted certificate authority certificates
```

### Failed Verification

If verification fails, DO NOT USE THE IMAGE. Possible causes:
- Image was not built from official repository
- Image was modified after signing
- Image source cannot be verified

## SBOM Verification

Software Bill of Materials documents are available in release artifacts.

Download SBOM:
```bash
# From GitHub release
wget https://github.com/tomtom215/cartographus/releases/download/v1.0.0/sbom-release-spdx.json
```

Inspect SBOM contents:
```bash
# View component list
jq '.packages[] | {name: .name, version: .versionInfo}' sbom-release-spdx.json

# View license summary
jq '.packages[].licenseConcluded' sbom-release-spdx.json | sort | uniq -c
```

## Security Policy

For security concerns or vulnerability reports, refer to the project's security policy in the repository.
