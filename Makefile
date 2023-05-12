.PHONY: clean
clean:
	@rm -rf dist
	@go clean -i

.PHONY: dist
dist:
	@pnpm build

.PHONY: fake_assets
fake_assets:
	@echo 'Skipping asset build'
	@mkdir -p dist
	@echo "assets build was skipped" > dist/index.html

.PHONY: test
test: fake_assets
	go test -cover ./...

.PHONY: build
build: dist
	CGO_ENABLED=0 go build -ldflags "-s -w"

.PHONY: docker
docker:
	@docker build -t amir20/dozzle .

.PHONY: dev
dev:
	pnpm dev

.PHONY: int
int:
	docker compose up --force-recreate --exit-code-from playwright
