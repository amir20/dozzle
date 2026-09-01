#!/usr/bin/env bash
#
# Creates one container that Dozzle will report as out of date, so the update
# flow can be watched end to end.
#
#   ./scripts/fake-update.sh          # set it up
#   ./scripts/fake-update.sh bump     # make it out of date again
#   ./scripts/fake-update.sh down     # tear it all down
#
# It runs a throwaway registry on port 5555 and pushes a new image to the same
# tag, so the container ends up running a digest the registry has moved past.
# Nothing outside this script is touched: no public images are retagged.

set -euo pipefail

REGISTRY_PORT=5555
REGISTRY_NAME=dozzle-fake-registry
IMAGE="localhost:${REGISTRY_PORT}/fake-app"
CONTAINER=fake-app

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
dim() { printf '\033[2m%s\033[0m\n' "$*"; }
bold() { printf '\033[1m%s\033[0m\n' "$*"; }

require_docker() {
  docker info >/dev/null 2>&1 || {
    red "docker is not running"
    exit 1
  }
}

# Builds and pushes a new image to :latest. Each call produces different
# content, so the digest changes every time.
push_image() {
  local version="$1"
  # The command reads the version out of the image at runtime rather than
  # baking it into CMD. Recreating a container preserves its explicit command,
  # so a baked-in marker would keep printing the old version even after a
  # successful update.
  docker build -q -t "${IMAGE}:latest" - >/dev/null <<DOCKERFILE
FROM busybox:latest
RUN echo "${version} (built $(date +%s%N))" > /version
CMD ["sh", "-c", "trap 'exit 0' TERM INT; while true; do echo \\"[fake-app] running image \$(cat /version)\\"; sleep 3 & wait \$!; done"]
DOCKERFILE
  docker push -q "${IMAGE}:latest" >/dev/null
}

start_registry() {
  if ! docker ps --filter "name=^${REGISTRY_NAME}$" --format '{{.Names}}' | grep -q .; then
    docker rm -f "$REGISTRY_NAME" >/dev/null 2>&1 || true
    docker run -d --name "$REGISTRY_NAME" -p "${REGISTRY_PORT}:5000" registry:2 >/dev/null
    dim "  waiting for the registry"
    for _ in $(seq 1 30); do
      curl -fsS "http://localhost:${REGISTRY_PORT}/v2/" >/dev/null 2>&1 && break
      sleep 0.5
    done
  fi
  green "  registry running on :${REGISTRY_PORT}"
}

up() {
  require_docker

  bold "starting a throwaway registry"
  start_registry

  bold "publishing v1 and running a container from it"
  push_image 1
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  docker run -d --name "$CONTAINER" "${IMAGE}:latest" >/dev/null
  green "  ${CONTAINER} is running v1"

  bump
}

bump() {
  require_docker
  start_registry >/dev/null

  bold "publishing a newer image to the same tag"
  push_image "$(date +%H%M%S)"
  green "  ${IMAGE}:latest now points at new content"

  echo
  bold "what to do next"
  cat <<INSTRUCTIONS

  1. open Dozzle and select the "${CONTAINER}" container
  2. the toolbar dot turns on and the menu says "Update available"
  3. click Update and watch it pull and recreate

  Dozzle needs actions enabled for the button:

    DOZZLE_ENABLE_ACTIONS=true go run .

  After updating, run '$0 bump' to make it out of date again.

INSTRUCTIONS
}

down() {
  require_docker

  bold "cleaning up"
  docker rm -f "$CONTAINER" >/dev/null 2>&1 && green "  removed ${CONTAINER}" || dim "  no container"
  docker rm -f "$REGISTRY_NAME" >/dev/null 2>&1 && green "  removed registry" || dim "  no registry"
  docker rmi "${IMAGE}:latest" >/dev/null 2>&1 || true
  green "  done"
}

case "${1:-up}" in
  up) up ;;
  bump) bump ;;
  down) down ;;
  *)
    echo "usage: $0 [up|bump|down]" >&2
    exit 1
    ;;
esac
