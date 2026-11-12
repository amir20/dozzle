go_test := "go test -cover -race -count 1 -timeout 40s ./..."

# List recipes
default:
    @just --list --unsorted

# Run backend and frontend with hot reload
dev: generate fake-assets
    bun run dev

# Several worktrees can each run an instance at once. Prints the URL it picked.
# Override any port by hand with DOZZLE_PORT / VITE_PORT / AGENT_PORT.

# Same as dev, on free ports derived from this checkout's path
dev-auto: generate fake-assets
    bun run dev:auto

# Run Go tests after go generate
test: fake-assets generate
    {{ go_test }}

# All generated code is committed, so CI does not need protoc installed just to
# reproduce identical output.

# Same as test, but skips go generate
test-ci: fake-assets certs
    {{ go_test }}

# Run the Playwright integration suite in docker compose
int:
    docker compose up --build --force-recreate --exit-code-from playwright

# Generate certificates, protobuf code and TypeScript types
generate: certs
    @go generate ./...

# Create the shared agent certificate and key if they are missing
certs:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ ! -f shared_key.pem ]; then
        openssl genpkey -algorithm Ed25519 -out shared_key.pem
        rm -f shared_cert.pem # a new key needs a new cert
    fi
    if [ ! -f shared_cert.pem ]; then
        openssl req -new -key shared_key.pem -out shared_request.csr -subj "/C=US/ST=California/L=San Francisco/O=Dozzle"
        openssl x509 -req -in shared_request.csr -signkey shared_key.pem -out shared_cert.pem -days 1825
        rm shared_request.csr
    fi

# Build frontend assets into dist/
dist:
    @bun run build

# Write a placeholder dist/ so Go can embed it without a frontend build
fake-assets:
    @echo 'Skipping asset build'
    @mkdir -p dist
    @echo "assets build was skipped" > dist/index.html

# Build the dozzle binary
build: dist generate
    CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/amir20/dozzle/internal/cli.Version=local"

# Build the amir20/dozzle:local image
docker: generate
    @docker build --build-arg TAG=local -t amir20/dozzle:local .

# Build the image and run it against the local docker socket
run: docker
    docker run -it --rm -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock -v /data amir20/dozzle:local

# Build, then serve the production assets
preview: build
    bun run preview

# Load the local image into an OrbStack VM and recreate its agent
agent-reload vm="dozzle-agent": docker
    #!/usr/bin/env bash
    set -uo pipefail
    echo "📦 Loading image into VM {{ vm }}..."
    docker save amir20/dozzle:local | orb exec -m {{ vm }} docker load
    echo "🔄 Recreating agent..."
    orb exec -m {{ vm }} docker stop dozzle-agent || true
    orb exec -m {{ vm }} docker rm dozzle-agent || true
    orb exec -m {{ vm }} docker run -d --name dozzle-agent -p 7007:7007 \
        -v /var/run/docker.sock:/var/run/docker.sock -v ~/dozzle-certs:/certs -v ~/dozzle-data:/data \
        -e DOZZLE_LEVEL=debug amir20/dozzle:local agent --cert /certs/shared_cert.pem --key /certs/shared_key.pem
    echo "✅ Agent reloaded"

# Remove build output and the shared certificates (generated code is committed and kept)
clean:
    @rm -rf dist
    @go clean -i
    @rm -f shared_key.pem shared_cert.pem
