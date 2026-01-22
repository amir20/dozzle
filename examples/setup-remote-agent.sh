#!/bin/bash
set -e

# Parse arguments
USE_LOCAL=false
POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
    --local)
      USE_LOCAL=true
      shift
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

if [ "$USE_LOCAL" = true ]; then
    DOZZLE_IMAGE="amir20/dozzle:local"
fi

echo "🚀 Setting up Dozzle Agent on OrbStack VM: $VM_NAME"
if [ "$USE_LOCAL" = true ]; then
    echo "   Using locally built image"
fi

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
if ! orb exec -m "$VM_NAME" bash -c "
set -e
docker run -d --name dozzle-agent \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ~/dozzle-certs:/certs \
  -p $AGENT_PORT:7007 \
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
echo "    amir20/dozzle:latest \\"
echo "    --remote-agent $VM_NAME.orb.local:$AGENT_PORT \\"
echo "    --cert /shared_cert.pem --key /shared_key.pem"
echo ""
echo "Or use environment variables in docker-compose.yml:"
echo ""
echo "  DOZZLE_REMOTE_AGENT: $VM_NAME.orb.local:$AGENT_PORT"
echo "  DOZZLE_CERT: /shared_cert.pem"
echo "  DOZZLE_KEY: /shared_key.pem"
echo ""
echo "Useful commands:"
echo "  View agent logs:   orb exec -m $VM_NAME docker logs -f dozzle-agent"
echo "  Stop agent:        orb exec -m $VM_NAME docker stop dozzle-agent"
echo "  Start agent:       orb exec -m $VM_NAME docker start dozzle-agent"
echo "  Remove agent:      orb exec -m $VM_NAME docker rm -f dozzle-agent"
echo "  Delete VM:         orb delete $VM_NAME"
