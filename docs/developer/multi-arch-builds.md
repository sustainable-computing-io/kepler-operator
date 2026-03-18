<!-- SPDX-FileCopyrightText: 2025 The Kepler Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Cross-Compilation Guide

Kepler Operator supports building for multiple architectures (amd64, arm64).
Since the operator uses `CGO_ENABLED=0`, Go handles cross-compilation natively
— no C cross-compiler or sysroot is needed.

## Building ARM64 Binaries

```bash
# Cross-compile binary for ARM64
make build-manager GOARCH=arm64
```

## Building Container Images

### Single-arch

```bash
# Build an ARM64 container image (loads into local Docker)
make operator-build GOARCH=arm64

# Build an amd64 container image (default)
make operator-build
```

### Multi-arch

Multi-arch builds use per-arch `docker build` followed by `docker manifest` to
create a manifest list. Images are built locally, then pushed and assembled
into a manifest in a separate step.

```bash
# Build multi-arch operator images locally (amd64 + arm64)
make operator-build-multi

# Push images and create multi-arch manifest
make operator-push-multi

# Same for bundle images
make bundle-build-multi
make bundle-push-multi

# Customize architectures
make operator-build-multi IMAGE_ARCHES="amd64 arm64 ppc64le"
```

### Full multi-arch build and deploy workflow

```bash
# Build multi-arch operator + generate bundle + build multi-arch bundle
make operator-build-multi bundle bundle-build-multi \
  IMG_BASE=quay.io/your-registry \
  VERSION=0.0.0-dev \
  KEPLER_IMG=quay.io/your-registry/kepler:latest

# Push all images and create manifests
make operator-push-multi bundle-push-multi \
  IMG_BASE=quay.io/your-registry \
  VERSION=0.0.0-dev

# Deploy via OLM
operator-sdk run bundle quay.io/your-registry/kepler-operator-bundle:0.0.0-dev \
  --install-mode AllNamespaces \
  --namespace operators
```

## How it works

The Dockerfile uses `FROM --platform=$BUILDPLATFORM` on the builder stage so the
Go compiler always runs natively on the host architecture. Cross-compilation is
achieved by passing `GOARCH=$TARGETARCH` to `go build`. This avoids QEMU
emulation entirely, making builds fast and reliable.

The `oc` CLI (needed for must-gather) is downloaded from the official OpenShift
mirror for the target architecture rather than pulled from a container image,
ensuring ARM64 support.

## Makefile Variables

| Variable       | Default        | Description                                   |
|----------------|----------------|-----------------------------------------------|
| `GOARCH`       | host arch      | Target architecture for single-arch builds    |
| `IMAGE_ARCHES` | `amd64 arm64`  | Architectures built by `*-multi` targets      |
