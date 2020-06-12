#COMMIT_HASH=`git rev-parse --short HEAD`
COMMIT_HASH=latest

FITTINGJOB_IMAGE=cyberagentoss/intelligent-hpa-fittingjob
CONTROLLER_IMAGE=cyberagentoss/intelligent-hpa-controller

fittingjob:
	docker build -t $(FITTINGJOB_IMAGE):$(COMMIT_HASH) ./fittingjob
	docker push $(FITTINGJOB_IMAGE):$(COMMIT_HASH)

controller:
	make -C ihpa-controller docker-build
	docker tag controller:latest $(CONTROLLER_IMAGE):$(COMMIT_HASH)
	docker push $(CONTROLLER_IMAGE):$(COMMIT_HASH)

.PHONY: fittingjob controller
