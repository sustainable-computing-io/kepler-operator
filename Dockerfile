# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

ARG VERSION
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build - cross-compile natively (CGO_ENABLED=0)
RUN make build-manager \
  GOARCH=${TARGETARCH} \
  VERSION=${VERSION} \
  GIT_COMMIT=${GIT_COMMIT} \
  GIT_BRANCH=${GIT_BRANCH}

FROM registry.access.redhat.com/ubi9-minimal:9.2
ARG TARGETARCH
RUN INSTALL_PKGS=" \
  rsync \
  tar \
  gzip \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all
RUN curl -sL "https://mirror.openshift.com/pub/openshift-v4/${TARGETARCH}/clients/ocp/stable/openshift-client-linux.tar.gz" \
    | tar -xzf - -C /usr/bin oc
WORKDIR /
COPY --from=builder /workspace/bin/manager .
COPY --from=builder /workspace/must-gather/* /usr/bin/
USER 65532:65532

ENTRYPOINT ["/manager"]
