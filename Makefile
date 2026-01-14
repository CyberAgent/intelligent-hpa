#COMMIT_HASH=`git rev-parse --short HEAD`
COMMIT_HASH=latest

# Default to GitHub Packages registry
REGISTRY?=ghcr.io/cyberagent
FITTINGJOB_IMAGE=$(REGISTRY)/intelligent-hpa/intelligent-hpa-fittingjob
CONTROLLER_IMAGE=$(REGISTRY)/intelligent-hpa/intelligent-hpa-controller

fittingjob:
	docker build -t $(FITTINGJOB_IMAGE):$(COMMIT_HASH) ./fittingjob
	docker push $(FITTINGJOB_IMAGE):$(COMMIT_HASH)

controller:
	make -C ihpa-controller docker-build
	docker tag controller:latest $(CONTROLLER_IMAGE):$(COMMIT_HASH)
	docker push $(CONTROLLER_IMAGE):$(COMMIT_HASH)

manifest:
	cd ihpa-controller && make manifests
	cd ihpa-controller/config/manager && kustomize edit set image controller=$(CONTROLLER_IMAGE):$(COMMIT_HASH)
	cd ihpa-controller && kustomize build config/default > ../manifests/intelligent-hpa.yaml

.PHONY: fittingjob controller manifest
