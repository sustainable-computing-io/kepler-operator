# The Dockerfile's resulting image is used in executing Kepler Operator e2e tests on ProwCI
FROM golang:1.21

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

# Cache dependencies before building source
RUN go mod download

# Install kubectl and oc
RUN curl -L -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xvzf oc.tar.gz \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/

# Copy the go source
COPY pkg/ pkg/
COPY tests/ tests/

# Compile test into e2e.test binary
RUN go test -c ./tests/e2e/
