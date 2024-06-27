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
	go test -cover -race ./...

.PHONY: build
build: dist
	CGO_ENABLED=0 go build -ldflags "-s -w"

.PHONY: docker
docker: shared_key.pem shared_cert.pem
	@docker build -t amir20/dozzle .

.PHONY: dev
dev:
	pnpm dev

.PHONY: int
int:
	docker compose up --build --force-recreate --exit-code-from playwright

shared_key.pem:
	@openssl genpkey -algorithm RSA -out shared_key.pem -pkeyopt rsa_keygen_bits:2048

shared_cert.pem:
	@openssl req -new -key shared_key.pem -out shared_request.csr -subj "/C=US/ST=California/L=San Francisco/O=Dozzle"
	@openssl x509 -req -in shared_request.csr -signkey shared_key.pem -out shared_cert.pem -days 365
	@rm shared_request.csr


.PHONY: push
push: shared_key.pem shared_cert.pem
	@docker build -t amir20/dozzle:agent .
	@docker push amir20/dozzle:agent
