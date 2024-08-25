REPOSITORY := ban11111
PROJECT := aurora-llm
REPOSITORY_PROJECT := $(REPOSITORY)/$(PROJECT)
TAG := v0.0.1

.PHONY: dev
dev:
	./scripts/run_dev.sh

.PHONY: image
image:
	docker build -t $(REPOSITORY_PROJECT):$(TAG) .
	@echo "Removing intermediate images"
	docker images --quiet --filter=dangling=true | xargs  docker rmi -f

.PHONY: push
push:
	docker push $(REPOSITORY_PROJECT):$(TAG)