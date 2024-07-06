PROTO_DIR := protos
GEN_DIR := internal/agent/pb
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GEN_FILES := $(patsubst $(PROTO_DIR)/%.proto,$(GEN_DIR)/%.pb.go,$(PROTO_FILES))

.PHONY: clean
clean:
	@rm -rf dist
	@go clean -i
	@rm -f shared_key.pem shared_cert.pem
	@rm -f $(GEN_DIR)/*.pb.go

.PHONY: dist
dist:
	@pnpm build

.PHONY: fake_assets
fake_assets:
	@echo 'Skipping asset build'
	@mkdir -p dist
	@echo "assets build was skipped" > dist/index.html

.PHONY: test
test: fake_assets generate
	go test -cover -race -count 1 -timeout 20s ./...

.PHONY: build
build: dist generate
	CGO_ENABLED=0 go build -ldflags "-s -w"

.PHONY: docker
docker: shared_key.pem shared_cert.pem
	@docker build -t amir20/dozzle .

generate: shared_key.pem shared_cert.pem $(GEN_FILES)

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

$(GEN_DIR)/%.pb.go: $(PROTO_DIR)/%.proto
	@go generate

.PHONY: push
push: docker
	@docker tag amir20/dozzle:latest amir20/dozzle:agent
	@docker push amir20/dozzle:agent
