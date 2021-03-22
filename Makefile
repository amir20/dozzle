TAG := $(shell git describe --tags)
PLATFROMS := linux/amd64,linux/arm/v7,linux/arm64/v8


.PHONY: publish
publish:
	docker buildx build --build-arg TAG=$(TAG) --platform $(PLATFROMS) -t amir20/dozzle:latest -t amir20/dozzle:$(TAG) --push .

.PHONY: clean
clean:
	@rm -rf static dozzle

static: $(shell find assets -type f)
ifdef SKIP_ASSET
	@echo 'Skipping yarn build'
	@mkdir -p static
	@touch static/index.html
else
	yarn build
endif

.PHONY: test
test: static
	go test -cover ./...

build: static
	CGO_ENABLED=0 go build -ldflags "-s -w"

int:
	docker-compose -f integration/docker-compose.test.yml up --build --force-recreate --exit-code-from integration
