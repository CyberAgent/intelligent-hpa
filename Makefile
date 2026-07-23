#COMMIT_HASH=`git rev-parse --short HEAD`
COMMIT_HASH=latest

# Default to GitHub Packages registry for this project
REGISTRY?=ghcr.io/cyberagent/intelligent-hpa
FITTINGJOB_IMAGE=$(REGISTRY)/intelligent-hpa-fittingjob
CONTROLLER_IMAGE=$(REGISTRY)/intelligent-hpa-controller

# Platforms for multi-arch builds
PLATFORMS ?= linux/amd64,linux/arm64

fittingjob:
	docker buildx build --platform $(PLATFORMS) -t $(FITTINGJOB_IMAGE):$(COMMIT_HASH) --push ./fittingjob

controller:
	docker buildx build --platform $(PLATFORMS) -t $(CONTROLLER_IMAGE):$(COMMIT_HASH) --push ./ihpa-controller

manifest:
	cd ihpa-controller && make manifests
	cd ihpa-controller/config/manager && kustomize edit set image controller=$(CONTROLLER_IMAGE):$(COMMIT_HASH)
	cd ihpa-controller && kustomize build config/default > ../manifests/intelligent-hpa.yaml

.PHONY: fittingjob controller manifest
