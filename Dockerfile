# Build the manager binary
FROM golang:1.24 AS builder

ARG VERSION
ARG GIT_COMMIT
ARG GIT_BRANCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN make build-manager \
  VERSION=${VERSION} \
  GIT_COMMIT=${GIT_COMMIT} \
  GIT_BRANCH=${GIT_BRANCH}

FROM quay.io/openshift/origin-cli:4.18 AS origincli

FROM registry.access.redhat.com/ubi9-minimal:9.2
RUN INSTALL_PKGS=" \
  rsync \
  tar \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all
WORKDIR /
COPY --from=builder /workspace/bin/manager .
COPY --from=builder /workspace/must-gather/* /usr/bin/
COPY --from=origincli /usr/bin/oc /usr/bin
USER 65532:65532

ENTRYPOINT ["/manager"]
