# Image URL to use all building/pushing image targets
IMG ?= cyberagentoss/intelligent-hpa-fittingjob:latest
# PLATFORMS defines the target platforms for multi-arch builds.
# Requirements:
# - docker buildx is available
# - BuildKit is enabled
# - the image can be pushed to your registry
PLATFORMS ?= linux/amd64,linux/arm64

# Build the docker image
docker-build:
	docker build . -t ${IMG}

# Build and push a multi-arch image
docker-buildx:
	docker buildx build --platform $(PLATFORMS) -t ${IMG} --push .

.PHONY: docker-build docker-buildx
