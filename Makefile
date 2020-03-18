TAG := $(shell git describe --tags)
PLATFROMS := linux/amd64,linux/arm64,linux/arm/v7

.PHONY: publish
publish:
	docker buildx build --build-arg TAG=$(TAG) --platform $(PLATFROMS) -t amir20/test:latest -t amir20/test:$(TAG) --push .
