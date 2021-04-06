TAG := $(shell git describe --tags)
PLATFROMS := linux/amd64,linux/arm/v7,linux/arm64/v8


.PHONY: publish
publish:
	docker buildx build --build-arg TAG=$(TAG) --platform $(PLATFROMS) -t amir20/dozzle:latest -t amir20/dozzle:$(TAG) --push .

.PHONY: clean
clean:
	@rm -rf static
	@go clean

.PHONY: static
static:
	@yarn build

.PHONY: fake_static
fake_static:
	@echo 'Skipping yarn build'
	@mkdir -p static
	@echo "yarn build was skipped" > static/index.html

.PHONY: test
test: fake_static
	go test -cover ./...

build: static
	CGO_ENABLED=0 go build -ldflags "-s -w"

dev:
	yarn dev

int:
	docker-compose -f integration/docker-compose.test.yml up --build --force-recreate --exit-code-from integration
