#!/usr/bin/env fish

# docker network create --driver bridge swarm-net

echo "Removing existing containers"
docker rm -f manager worker-1 worker-2

echo "Creating manager"
docker run -d --name manager \
    --privileged \
    --network swarm-net \
    --network-alias manager \
    -p 2377:2377 \
    -p 7946:7946 \
    -p 4789:4789 \
    -p 8000-9000:8000-9000 \
    -v ./examples:/examples \
    docker

# Store join command in a variable
sleep 2
echo "Initializing swarm"
set JOIN_COMMAND (docker exec manager docker swarm init | grep "swarm join --token")

echo "Creating workers"
for i in 1 2
    docker run -d --name worker-$i \
        --privileged \
        --network swarm-net \
        --network-alias worker-$i \
        docker
end

sleep 2

for i in 1 2
    echo "Joining worker-$i to swarm"
    docker exec worker-$i sh -c "$JOIN_COMMAND"
end

function cleanup
    docker rm -f manager worker-1 worker-2
end

function swarm
    docker exec manager docker $argv
end
