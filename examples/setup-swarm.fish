#!/usr/bin/env fish

echo "🧹 Removing existing containers and network"
docker rm -f manager worker-1 worker-2 2>/dev/null
docker network rm swarm-net 2>/dev/null

echo "🌐 Creating bridge network"
docker network create --driver bridge swarm-net

echo "🖥️ Creating manager"
docker run -d --name manager \
    --privileged \
    --network swarm-net \
    --network-alias manager \
    -p 2377:2377 \
    -p 7946:7946 \
    -p 4789:4789 \
    -p 8090:8090 \
    -v ./examples:/examples \
    docker

# Wait for dockerd to be ready inside manager
echo "⏳ Waiting for manager dockerd to start..."
while not docker exec manager docker info >/dev/null 2>&1
    sleep 1
end

echo "🐝 Initializing swarm"
set MANAGER_IP (docker exec manager hostname -i | string trim)
set JOIN_TOKEN (docker exec manager docker swarm init --advertise-addr $MANAGER_IP 2>&1 | string match -r 'SWMTKN-\S+')
if test -z "$JOIN_TOKEN"
    echo "❌ Failed to get swarm join token"
    exit 1
end

echo "👷 Creating workers"
for i in 1 2
    docker run -d --name worker-$i \
        --privileged \
        --network swarm-net \
        --network-alias worker-$i \
        docker
end

# Wait for worker dockerd to be ready
for i in 1 2
    echo "⏳ Waiting for worker-$i dockerd to start..."
    while not docker exec worker-$i docker info >/dev/null 2>&1
        sleep 1
    end
end

for i in 1 2
    echo "🔗 Joining worker-$i to swarm"
    docker exec worker-$i docker swarm join --token $JOIN_TOKEN $MANAGER_IP:2377
end

echo "✅ Swarm is ready. Deploying Dozzle..."

echo "🔨 Building local image..."
env CLOUD_URL=http://localhost:3000 make -C (git rev-parse --show-toplevel) docker

echo "📦 Loading image into swarm nodes..."
for node in manager worker-1 worker-2
    docker save amir20/dozzle:local | docker exec -i $node docker load
end

# Create the stack file inside the manager
docker exec manager sh -c 'cat > /dozzle-stack.yml << "EOF"
services:
  dozzle:
    image: amir20/dozzle:local
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_ENABLE_ACTIONS=true
      - DOLIGENCE_URL=http://doligence-api:8080
      - AGENT_URL=http://doligence-api:8082
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8090:8080
    networks:
      - dozzle
    # 172.16.48.100 is a static IP for the doligence-api dev container,
    # assigned via doligence/compose.override.yaml on the swarm-net
    # external network (the network this script creates). Local-dev only.
    extra_hosts:
      - "doligence-api:172.16.48.100"
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
volumes:
  dozzle-data:
EOF'

docker exec manager docker stack deploy -c /dozzle-stack.yml dozzle
echo "🚀 Dozzle deployed! Access at http://localhost:8090"

function swarm-cleanup
    docker exec manager docker stack rm dozzle 2>/dev/null
    docker rm -f manager worker-1 worker-2
    docker network rm swarm-net 2>/dev/null
    functions -e swarm-cleanup
    functions -e swarm
    functions -e swarm-deploy
end

function swarm
    docker exec manager docker $argv
end

function swarm-deploy
    echo "🔨 Building local image..."
    env CLOUD_URL=http://localhost:3000 make -C (git rev-parse --show-toplevel) docker
    echo "📦 Loading image into swarm nodes..."
    for node in manager worker-1 worker-2
        docker save amir20/dozzle:local | docker exec -i $node docker load
    end
    echo "🚀 Redeploying stack..."
    docker exec manager docker stack rm dozzle 2>/dev/null
    sleep 2
    docker exec manager docker stack deploy -c /dozzle-stack.yml dozzle
    echo "✅ Deployed local changes!"
end

echo "💡 Functions 'swarm', 'swarm-deploy', and 'swarm-cleanup' are available in this session."
