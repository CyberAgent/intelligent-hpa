
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Options for controller-gen (v1 CRDs)
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Version of controller-gen (keep in sync with .github/workflows/build-and-push.yaml)
CONTROLLER_TOOLS_VERSION ?= v0.22.0

# Version of golangci-lint (keep in sync with .github/workflows/lint.yaml)
GOLANGCI_LINT_VERSION ?= v2.13.1

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager cmd/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./cmd/main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply --server-side --force-conflicts --field-manager=ihpa-controller -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy:
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply --server-side --force-conflicts --field-manager=ihpa-controller -f -

# Undeploy controller from the configured Kubernetes cluster in ~/.kube/config
# Call with ignore-not-found=true to ignore resource not found errors during deletion.
ifndef ignore-not-found
  ignore-not-found = false
endif
undeploy:
	kustomize build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

# Run golangci-lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

# Run golangci-lint and apply fixes
lint-fix: golangci-lint
	$(GOLANGCI_LINT) run --fix

# Verify the golangci-lint configuration
lint-config: golangci-lint
	$(GOLANGCI_LINT) config verify

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Build and push a multi-arch image
PLATFORMS ?= linux/amd64,linux/arm64
docker-buildx: test
	docker buildx build --platform $(PLATFORMS) -t ${IMG} --push .

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
	@if [ "$$($(CONTROLLER_GEN) --version 2>/dev/null | cut -d' ' -f2)" != "$(CONTROLLER_TOOLS_VERSION)" ]; then \
		GOBIN=$(GOBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION); \
	fi
CONTROLLER_GEN=$(GOBIN)/controller-gen

# find or download golangci-lint
# download golangci-lint if necessary
golangci-lint:
	@if [ "$$($(GOLANGCI_LINT) version --short 2>/dev/null)" != "$(GOLANGCI_LINT_VERSION:v%=%)" ]; then \
		GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@test -f .custom-gcl.yml && { \
		echo "Building custom golangci-lint with plugins..." && \
		$(GOLANGCI_LINT) custom --destination $(GOBIN) --name golangci-lint-custom && \
		mv -f $(GOBIN)/golangci-lint-custom $(GOLANGCI_LINT); \
	} || true
GOLANGCI_LINT=$(GOBIN)/golangci-lint

.PHONY: all test manager run install uninstall deploy undeploy manifests fmt vet generate docker-build docker-buildx docker-push controller-gen lint lint-fix lint-config golangci-lint
