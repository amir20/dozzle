#!/bin/bash
set -e

usage() {
  cat <<'EOF'
Usage: ./setup-remote-agent.sh [options] [VM_NAME] [DISTRO] [AGENT_PORT]

Creates an OrbStack VM, installs Docker and runs a Dozzle agent in it.

Positional arguments:
  VM_NAME       OrbStack VM to create        (default: dozzle-agent)
  DISTRO        Distro for the VM            (default: ubuntu)
  AGENT_PORT    Host port for the agent      (default: 7007)

Options:
  --local             Build and load amir20/dozzle:local instead of pulling :latest
  --agent-url URL     Dozzle Cloud gRPC endpoint the agent connects to
                      (default: https://agent.doligence.dozzle.dev)
  --cloud-url URL     Dozzle Cloud HTTP API the agent posts notifications to
                      (default: https://doligence.dozzle.dev)
  -h, --help          Show this help

Both URLs can also come from the environment, as AGENT_URL and DOLIGENCE_URL —
the same names the agent itself reads — so a shell that already exports them
needs no flags. Flags win over the environment.

Pointing an agent at a local Dozzle Cloud stack:

  ./setup-remote-agent.sh --local \
    --agent-url http://host.orb.internal:8082 \
    --cloud-url http://host.orb.internal:8080

host.orb.internal is the Mac host, and 8082/8080 are the gRPC and public ports
compose.override.yaml publishes there. An http:// URL makes the agent dial gRPC
in plaintext, which is what the local stack serves.

Use the name, not OrbStack's 198.19.249.2 host address: that one is only routable
from a Docker container running under OrbStack directly. The agent this script
sets up runs inside a Linux VM, which reaches the Mac over its own bridge
instead, and dialling 198.19.249.2 from there times out with nothing to explain
why. The name resolves from both.

Setting these does not by itself connect the agent to Dozzle Cloud: it dials
only once it has an API key, which arrives when the main Dozzle instance pushes
its cloud config down and the agent persists it to /data/cloud.yml.
EOF
}

# Parse arguments
USE_LOCAL=false
POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
    --local)
      USE_LOCAL=true
      shift
      ;;
    --agent-url)
      AGENT_URL="$2"
      shift 2
      ;;
    --cloud-url)
      DOLIGENCE_URL="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      POSITIONAL_ARGS+=("$1")
      shift
      ;;
  esac
done

# Configuration
VM_NAME="${POSITIONAL_ARGS[0]:-dozzle-agent}"
DISTRO="${POSITIONAL_ARGS[1]:-ubuntu}"
AGENT_PORT="${POSITIONAL_ARGS[2]:-7007}"
SHARED_CERT="./shared_cert.pem"
SHARED_KEY="./shared_key.pem"
DOZZLE_IMAGE="amir20/dozzle:latest"

# Dozzle Cloud endpoints. These are the agent's own env vars, passed through
# rather than reinvented: AGENT_URL is the gRPC endpoint it dials for tool
# dispatch and log streaming, DOLIGENCE_URL is the HTTP API its cloud
# dispatcher posts notifications to. They are separate on purpose — setting
# only one leaves the other half of the agent talking to production.
CLOUD_AGENT_URL="${AGENT_URL:-https://agent.doligence.dozzle.dev}"
CLOUD_API_URL="${DOLIGENCE_URL:-https://doligence.dozzle.dev}"

if [ "$USE_LOCAL" = true ]; then
    DOZZLE_IMAGE="amir20/dozzle:local"
fi

echo "🚀 Setting up Dozzle Agent on OrbStack VM: $VM_NAME"
if [ "$USE_LOCAL" = true ]; then
    echo "   Using locally built image"
fi
echo "   Dozzle Cloud gRPC: $CLOUD_AGENT_URL"
echo "   Dozzle Cloud API:  $CLOUD_API_URL"

# Verify shared certificates exist
if [ ! -f "$SHARED_CERT" ]; then
    echo "❌ Shared certificate not found at $SHARED_CERT"
    echo "   Run 'make generate' to create certificates"
    exit 1
fi

if [ ! -f "$SHARED_KEY" ]; then
    echo "❌ Shared key not found at $SHARED_KEY"
    echo "   Run 'make generate' to create certificates"
    exit 1
fi

echo "✅ Found shared certificates"

# Step 1: Create the VM
echo "📦 Creating VM..."
if orb list | grep -q "^$VM_NAME"; then
    echo "⚠️  VM $VM_NAME already exists. Delete it first with: orb delete $VM_NAME"
    exit 1
fi

orb create "$DISTRO" "$VM_NAME"
echo "✅ VM created"

# Wait for VM to be ready
sleep 3

# Step 2: Install Docker in the VM
echo "🐳 Installing Docker..."
if ! orb exec -m "$VM_NAME" bash -c 'curl -fsSL https://get.docker.com | sh && sudo usermod -aG docker $(whoami)'; then
    echo "❌ Docker installation failed"
    exit 1
fi

echo "✅ Docker installed"

# Step 3: Copy shared certificates to VM
echo "🔐 Copying shared certificates to VM..."
orb exec -m "$VM_NAME" bash -c 'mkdir -p ~/dozzle-certs'

echo "  Copying shared_cert.pem..."
cat "$SHARED_CERT" | orb exec -m "$VM_NAME" bash -c 'cat > ~/dozzle-certs/shared_cert.pem'

echo "  Copying shared_key.pem..."
cat "$SHARED_KEY" | orb exec -m "$VM_NAME" bash -c 'cat > ~/dozzle-certs/shared_key.pem'

echo "✅ Certificates copied"

# Step 4: Load or pull Dozzle image
if [ "$USE_LOCAL" = true ]; then
    echo "🔨 Building local Docker image..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

    if ! (cd "$PROJECT_ROOT" && make docker); then
        echo "❌ Failed to build Docker image"
        exit 1
    fi

    echo "📦 Loading image into VM..."
    if ! docker save amir20/dozzle:local | orb exec -m "$VM_NAME" docker load; then
        echo "❌ Failed to load image into VM"
        exit 1
    fi
    echo "✅ Local image loaded"
else
    echo "📥 Pulling Dozzle image..."
    if ! orb exec -m "$VM_NAME" docker pull amir20/dozzle:latest; then
        echo "❌ Failed to pull image"
        exit 1
    fi
fi

# Step 5: Start Dozzle agent
echo "🎯 Starting Dozzle agent..."
orb exec -m "$VM_NAME" bash -c 'mkdir -p ~/dozzle-data'
if ! orb exec -m "$VM_NAME" bash -c "
set -e
docker run -d --name dozzle-agent \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ~/dozzle-certs:/certs \
  -v ~/dozzle-data:/data \
  -p $AGENT_PORT:7007 \
  -e DOZZLE_LEVEL=debug \
  -e AGENT_URL=$CLOUD_AGENT_URL \
  -e DOLIGENCE_URL=$CLOUD_API_URL \
  $DOZZLE_IMAGE agent \
  --cert /certs/shared_cert.pem \
  --key /certs/shared_key.pem
"; then
    echo "❌ Failed to start Dozzle agent"
    exit 1
fi

echo "✅ Dozzle agent started"

# Step 6: Wait for agent to be ready
echo "⏳ Waiting for agent to be ready..."
sleep 3

# Step 7: Verify agent is running
echo "🧪 Verifying agent is running..."
if orb exec -m "$VM_NAME" docker ps --filter name=dozzle-agent --format "{{.Status}}" | grep -q "Up"; then
    echo "✅ Agent is running"
else
    echo "❌ Agent failed to start. Check logs with: orb exec -m $VM_NAME docker logs dozzle-agent"
    exit 1
fi

# Print usage instructions
echo ""
echo "🎉 Setup complete!"
echo ""
echo "Dozzle agent is running on:"
echo "  $VM_NAME.orb.local:$AGENT_PORT"
echo ""
echo "To connect from your Dozzle instance, add this remote agent:"
echo ""
echo "  docker run -v /var/run/docker.sock:/var/run/docker.sock \\"
echo "    -v $PWD/shared_cert.pem:/shared_cert.pem:ro \\"
echo "    -v $PWD/shared_key.pem:/shared_key.pem:ro \\"
echo "    -p 8080:8080 \\"
echo "    -e AGENT_URL=$CLOUD_AGENT_URL \\"
echo "    -e DOLIGENCE_URL=$CLOUD_API_URL \\"
echo "    amir20/dozzle:latest \\"
echo "    --remote-agent $VM_NAME.orb.local:$AGENT_PORT \\"
echo "    --cert /shared_cert.pem --key /shared_key.pem"
echo ""
echo "Or use environment variables in docker-compose.yml:"
echo ""
echo "  DOZZLE_REMOTE_AGENT: $VM_NAME.orb.local:$AGENT_PORT"
echo "  DOZZLE_CERT: /shared_cert.pem"
echo "  DOZZLE_KEY: /shared_key.pem"
echo "  AGENT_URL: $CLOUD_AGENT_URL"
echo "  DOLIGENCE_URL: $CLOUD_API_URL"
echo ""
echo "Useful commands:"
echo "  View agent logs:   orb exec -m $VM_NAME docker logs -f dozzle-agent"
echo "  Stop agent:        orb exec -m $VM_NAME docker stop dozzle-agent"
echo "  Start agent:       orb exec -m $VM_NAME docker start dozzle-agent"
echo "  Remove agent:      orb exec -m $VM_NAME docker rm -f dozzle-agent"
echo "  Delete VM:         orb delete $VM_NAME"
