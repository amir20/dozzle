.PHONY: clean
clean:
	@rm -rf static
	@go clean -i

.PHONY: static
static:
	@pnpm build

.PHONY: fake_static
fake_static:
	@echo 'Skipping asset build'
	@mkdir -p static
	@echo "assets build was skipped" > static/index.html

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
	pnpm dev

.PHONY: int
int:
	docker-compose -f integration/docker-compose.test.yml up --build --force-recreate --exit-code-from integration
