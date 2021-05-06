.PHONY: clean
clean:
	@rm -rf static
	@go clean -i

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

.PHONY: build
build: static
	CGO_ENABLED=0 go build -ldflags "-s -w"

.PHONY: docker
docker:
	@docker build -t amir20/dozzle .

.PHONY: dev
dev:
	yarn dev

.PHONY: int
int:
	docker-compose -f integration/docker-compose.test.yml up --build --force-recreate --exit-code-from integration
