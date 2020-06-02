TAG := $(shell git describe --tags)
PLATFROMS := linux/amd64,linux/arm64/v8

.PHONY: publish
publish:
	docker buildx build --build-arg TAG=$(TAG) --platform $(PLATFROMS) -t amir20/dozzle:latest -t amir20/dozzle:$(TAG) --push .
