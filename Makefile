# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

THIS_FILE := $(lastword $(MAKEFILE_LIST))

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# WEBHOOKS
ENABLE_WEBHOOKS ?= true # enable webhooks by default

# Setting GOENV
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= $(shell cat VERSION)

BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

KEPLER_VERSION ?=release-0.7.12
KEPLER_REBOOT_VERSION ?=v0.0.6

# IMG_BASE and KEPLER_IMG_BASE are set to distinguish between Operator-specific images and Kepler-Specific images.
# IMG_BASE is used for building and pushing operator related images.
# KEPLER_IMG_BASE is exclusively used for Kepler related images.
# This separation ensures that local development and deployment of operator images do not interfere with Kepler images.
IMG_BASE ?= quay.io/sustainable_computing_io
KEPLER_IMG_BASE ?= quay.io/sustainable_computing_io/kepler
KEPLER_REBOOT_IMG_BASE ?= quay.io/sustainable_computing_io/kepler-reboot

# OPERATOR_IMG define the image:tag used for the operator
# You can use it as an arg. (E.g make operator-build OPERATOR_IMG=<some-registry>:<version>)
OPERATOR_IMG ?= $(IMG_BASE)/kepler-operator:$(VERSION)
ADDITIONAL_TAGS ?=

KEPLER_IMG ?= $(KEPLER_IMG_BASE):$(KEPLER_VERSION)
KEPLER_REBOOT_IMG ?= $(KEPLER_REBOOT_IMG_BASE):$(KEPLER_REBOOT_VERSION)

# E2E_TEST_IMG defines the image:tag used for the e2e test image
E2E_TEST_IMG ?=$(IMG_BASE)/kepler-operator-e2e:$(VERSION)

LDFLAGS=-ldflags "\
	-X github.com/sustainable.computing.io/kepler-operator/pkg/version.version=$(VERSION) \
	-X github.com/sustainable.computing.io/kepler-operator/pkg/version.buildTime=$(BUILD_TIME) \
	-X github.com/sustainable.computing.io/kepler-operator/pkg/version.gitBranch=$(GIT_BRANCH) \
	-X github.com/sustainable.computing.io/kepler-operator/pkg/version.gitCommit=$(GIT_COMMIT) \
"

.PHONY: fresh
fresh: ## default target - sets up a k8s cluster with images ready for deployment
	@$(MAKE) -f $(THIS_FILE) \
		cluster-restart \
		operator-build operator-push \
		bundle bundle-build \
		bundle-push \
		IMG_BASE=localhost:5001 VERSION=0.0.0-dev ;\

	@echo -e '\n        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'
	@echo -e ' 🎊  Operator has been successfully built and deployed! 🎊 \n'
	@kubectl cluster-info
	@echo -e '        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'
	@echo -e ' 🔔 Next step see kepler in action:'
	@echo -e '    ❯   ./tmp/bin/operator-sdk run bundle localhost:5001/kepler-operator-bundle:0.0.0-dev \ '
	@echo -e '         --install-mode AllNamespaces --namespace operators --skip-tls '
	@echo -e '    ❯ kubectl apply -f config/samples/kepler.system_v1alpha1_kepler.yaml \n'
	@echo -e '        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'


.PHONY: all
all: operator-build bundle bundle-build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(THIS_FILE)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: shfmt ## Run go fmt against code.
	go fmt ./...
	PATH=./tmp/bin:$$PATH \
		shfmt -l -w ./**/*.sh \
			./must-gather/gather*

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test:  fmt vet  ## Run tests.
		go test ./pkg/... -coverprofile cover.out

.PHONY: coverage
coverage: test ## Run tests and generate coverage report.
	go tool cover -html=cover.out -o cover.html

.PHONY: docs
docs: crdoc manifests ## Generate docs.
	$(CRDOC) --resources config/crd/bases --output docs/api.md

##@ Development env
CLUSTER_PROVIDER ?= kind
LOCAL_DEV_CLUSTER_VERSION ?= main
GRAFANA_ENABLE ?= false
PROMETHEUS_ENABLE ?= false
KIND_WORKER_NODES ?=2

.PHONY: cluster-up
cluster-up: ## setup a cluster for local development
	CLUSTER_PROVIDER=$(CLUSTER_PROVIDER) \
	VERSION=$(LOCAL_DEV_CLUSTER_VERSION) \
	GRAFANA_ENABLE=$(GRAFANA_ENABLE) \
	PROMETHEUS_ENABLE=$(PROMETHEUS_ENABLE) \
	KIND_WORKER_NODES=$(KIND_WORKER_NODES) \
	./hack/cluster.sh up

.PHONY: cluster-prereqs
cluster-prereqs: ## setup a cluster prerequisites for local operator development
	./hack/cluster.sh prereqs

.PHONY: cluster-restart
cluster-restart: ## restart the local development cluster
	CLUSTER_PROVIDER=$(CLUSTER_PROVIDER) \
	VERSION=$(LOCAL_DEV_CLUSTER_VERSION) \
	GRAFANA_ENABLE=$(GRAFANA_ENABLE) \
	PROMETHEUS_ENABLE=$(PROMETHEUS_ENABLE) \
	KIND_WORKER_NODES=$(KIND_WORKER_NODES) \
	./hack/cluster.sh restart

.PHONY: cluster-down
cluster-down: ## delete the local development cluster
	CLUSTER_PROVIDER=$(CLUSTER_PROVIDER) \
	VERSION=$(LOCAL_DEV_CLUSTER_VERSION) \
	./hack/cluster.sh down

##@ Build

.PHONY: build
build: manifests generate ## Build manager binary.
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/manager ./cmd/...

OPENSHIFT ?= true
RUN_ARGS ?=

.PHONY: run
run: install fmt vet ## Run a controller from your host against openshift cluster
	go run ./cmd/... \
		--kepler.image=$(KEPLER_IMG) \
		--kepler-reboot.image=$(KEPLER_REBOOT_IMG) \
		--zap-devel --zap-log-level=8 \
		--openshift=$(OPENSHIFT) \
		$(RUN_ARGS) \
		2>&1 | tee tmp/operator.log

# docker_tag accepts an image:tag and a list of additional tags comma-separated
# it tags the image with the additional tags
# E.g. given foo:bar, a,b,c, it will tag foo:bar as  foo:a, foo:b, foo:c
define docker_tag
@{ \
	set -eu ;\
	img="$(1)" ;\
	tags="$(2)" ;\
	echo "tagging container image $$img with additional tags: '$$tags'" ;\
	\
	img_path=$${img%:*} ;\
	for tag in $$(echo $$tags | tr -s , ' ' ); do \
		docker tag $$img $$img_path:$$tag ;\
	done \
}
endef


# docker_push accepts an image:tag and a list of additional tags comma-separated
# it push the image:tag all other images with the additional tags
# E.g. given foo:bar, a,b,c, it will push foo:bar, foo:a, foo:b, foo:c
define docker_push
@{ \
	set -eu ;\
	img="$(1)" ;\
	tags="$(2)" ;\
	echo "docker push $$img and additional tags: '$$tags'" ;\
	\
	img_path=$${img%:*} ;\
	docker push $$img ;\
	for tag in $$(echo $$tags | tr -s , ' ' ); do \
		docker push $$img_path:$$tag ;\
	done \
}
endef


# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: operator-build
operator-build: manifests generate test ## Build docker image with the manager.
	go mod tidy
	docker build -t $(OPERATOR_IMG) \
		--platform=linux/$(GOARCH) .
	$(call docker_tag,$(OPERATOR_IMG),$(ADDITIONAL_TAGS))


.PHONY: operator-push
operator-push: ## Push docker image with the manager.
	$(call docker_push,$(OPERATOR_IMG),$(ADDITIONAL_TAGS))


.PHONY: e2e-test-image
e2e-test-image: test
	docker build -f tests/Dockerfile \
		--platform=linux/$(GOARCH) \
		-t $(E2E_TEST_IMG) .

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: kustomize generate manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | \
		kubectl apply --server-side --force-conflicts -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | \
		kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: install ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default/k8s | \
		sed  -e "s|<OPERATOR_IMG>|$(OPERATOR_IMG)|g" \
		     -e "s|<KEPLER_IMG>|$(KEPLER_IMG)|g" \
		     -e "s|<KEPLER_REBOOT_IMG>|$(KEPLER_REBOOT_IMG)|g" \
		| tee tmp/deploy.yaml | \
		kubectl apply --server-side --force-conflicts -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default/k8s | \
		kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location where binaries are installed
LOCALBIN ?= $(shell pwd)/tmp/bin

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CRDOC ?= $(LOCALBIN)/crdoc

# NOTE: please keep this list sorted so that it can be easily searched
TOOLS = controller-gen \
		crdoc \
		govulncheck \
		jq \
		kubectl \
		kustomize \
		oc \
		operator-sdk \
		shfmt \
		yq \

.PHONY: tools
tools:
	./hack/tools.sh

$(TOOLS):
	./hack/tools.sh $@

mod-tidy:
	@go mod tidy

check-govuln: govulncheck mod-tidy
	@govulncheck ./... || true

escapes_detect: mod-tidy
	@go build -tags $(GO_BUILD_TAGS) -gcflags="-m -l" ./... 2>&1 | grep "escapes to heap" || true


CHANNELS ?=alpha

##@ OLM bundle
#
# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
DEFAULT_CHANNEL ?= alpha
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>:<version>)
BUNDLE_IMG ?= $(IMG_BASE)/kepler-operator-bundle:$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# VERSION_REPLACED defines the version replaced in the bundle
VERSION_REPLACED ?=

.PHONY: bundle
bundle: generate manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	OPERATOR_IMG=$(OPERATOR_IMG) \
	KEPLER_IMG=$(KEPLER_IMG) \
	KEPLER_REBOOT_IMG=$(KEPLER_REBOOT_IMG) \
	VERSION=$(VERSION) \
	VERSION_REPLACED=$(VERSION_REPLACED) \
	BUNDLE_GEN_FLAGS='$(BUNDLE_GEN_FLAGS)' \
		hack/bundle.sh

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile \
		-t $(BUNDLE_IMG) \
		--platform=linux/$(GOARCH) .
	$(call docker_tag,$(BUNDLE_IMG),$(ADDITIONAL_TAGS))

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(call docker_push,$(BUNDLE_IMG),$(ADDITIONAL_TAGS))

.PHONY: create-bundle
create-bundle:
	./hack/bundle.sh

##@ OLM Catalog

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMG_BASE)/kepler-operator-catalog:$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	docker push $(CATALOG_IMG)
