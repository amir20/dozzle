#!/usr/bin/env bash
#
# Creates a set of containers that between them light up every element of the
# container title bar: the name dropdown, health, links, volume warnings, the
# image tag, and the CPU/MEM/NET/DISK strip.
#
#   ./scripts/fake-title-bar.sh          # set it up
#   ./scripts/fake-title-bar.sh down     # tear it all down
#
# Every container is named fake-tb-* and is removed by `down`. Nothing outside
# that prefix is touched. Image update alerts are a separate flow, see
# ./scripts/fake-update.sh.

set -euo pipefail

PREFIX=fake-tb
NETWORK="${PREFIX}-net"
VOLUME="${PREFIX}-vol"
# A long, private-looking tag so the title bar has something to truncate. Built
# locally from busybox; `down` removes it again.
LONG_IMAGE="registry.internal.example.com/platform/team-observability/payments-api-worker:v2.14.7-rc3"

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

# Chatty on stdout and stderr both, so the stream dots in the toolbar have
# something to filter and the log body is never empty.
noisy_cmd() {
  cat <<'CMD'
trap 'exit 0' TERM INT
i=0
while true; do
  i=$((i + 1))
  echo "[$(date +%T)] request id=$i path=/api/v1/orders status=200 duration=${i}ms"
  if [ $((i % 5)) -eq 0 ]; then
    echo "[$(date +%T)] level=error upstream timeout after 30s, retrying" >&2
  fi
  if [ $((i % 7)) -eq 0 ]; then
    echo "{\"level\":\"info\",\"msg\":\"checkpoint\",\"seq\":$i,\"service\":\"payments\"}"
  fi
  sleep 2 & wait $!
done
CMD
}

run() {
  local name="$1"
  shift
  docker rm -f "${PREFIX}-${name}" >/dev/null 2>&1 || true
  docker run -d --name "${PREFIX}-${name}" "$@" >/dev/null
  green "  ${PREFIX}-${name}"
}

build_long_image() {
  docker build -q -t "$LONG_IMAGE" - >/dev/null <<'DOCKERFILE'
FROM busybox:latest
DOCKERFILE
}

up() {
  require_docker

  bold "pulling base images"
  docker pull -q busybox:latest >/dev/null
  docker pull -q alpine:latest >/dev/null
  dim "  busybox, alpine"

  bold "building the long-named image"
  build_long_image
  dim "  ${LONG_IMAGE}"

  docker network create "$NETWORK" >/dev/null 2>&1 || true
  docker volume create "$VOLUME" >/dev/null 2>&1 || true

  # --- name dropdown: three containers sharing one dev.dozzle.name ----------
  # Dozzle groups these under a single title with a count badge. One is left
  # exited so the dropdown shows a red status dot and a "finished" timestamp.
  bold "three containers sharing a name (dropdown + count badge)"
  run api-1 \
    --label dev.dozzle.name=payments-api \
    --label dev.dozzle.group=platform \
    --cpus 0.5 --memory 256m \
    "$LONG_IMAGE" sh -c "$(noisy_cmd)"
  run api-2 \
    --label dev.dozzle.name=payments-api \
    --label dev.dozzle.group=platform \
    --cpus 0.25 --memory 128m \
    "$LONG_IMAGE" sh -c "$(noisy_cmd)"
  docker rm -f "${PREFIX}-api-3" >/dev/null 2>&1 || true
  docker run -d --name "${PREFIX}-api-3" \
    --label dev.dozzle.name=payments-api \
    --label dev.dozzle.group=platform \
    busybox:latest sh -c 'echo "starting"; sleep 2; echo "crashed"; exit 1' >/dev/null
  green "  ${PREFIX}-api-3 (exits on its own, shows as stopped in the dropdown)"

  # --- health states -------------------------------------------------------
  bold "health states (green check, red cross, neutral starting)"
  run healthy \
    --health-cmd 'true' --health-interval 5s \
    alpine:latest sh -c "$(noisy_cmd)"
  run unhealthy \
    --health-cmd 'false' --health-interval 5s --health-retries 1 \
    alpine:latest sh -c "$(noisy_cmd)"
  # A start period longer than you will ever wait keeps this one "starting".
  run starting \
    --health-cmd 'sleep 3600' --health-interval 30s --health-start-period 24h \
    alpine:latest sh -c "$(noisy_cmd)"

  # --- container links -----------------------------------------------------
  bold "container links"
  run linked \
    --label dev.dozzle.url=https://dozzle.dev \
    alpine:latest sh -c "$(noisy_cmd)"
  dim "    dev.dozzle.url set: renders the open-in-new icon"
  run link-hint-port \
    -p 18081:80 \
    alpine:latest sh -c "$(noisy_cmd)"
  dim "    published port, no url: offers the link hint"
  run link-hint-traefik \
    -p 18082:80 \
    --label traefik.enable=true \
    --label 'traefik.http.routers.orders.rule=Host(`orders.example.test`)' \
    --label traefik.http.routers.orders.tls.certresolver=le \
    --label 'traefik.http.routers.orders-admin.rule=Host(`admin.orders.example.test`)' \
    alpine:latest sh -c "$(noisy_cmd)"
  dim "    traefik labels: the hint offers several suggestions to pick from"

  # --- volumes -------------------------------------------------------------
  # The badge only appears when the *Dozzle host* sees the mount source at 85%
  # or more, so the bind mount below reports whatever your disk really is.
  bold "volumes"
  run volumes \
    -v "$PWD:/app:ro" \
    -v "${VOLUME}:/data" \
    alpine:latest sh -c "$(noisy_cmd)"
  dim "    a bind mount (real usage) and a named volume"

  # --- busy: fills the NET / DISK rows and the CPU / MEM sparklines ---------
  bold "a busy container (NET and DISK rows, moving sparklines)"
  run busy \
    --network "$NETWORK" \
    --cpus 1 --memory 512m \
    alpine:latest sh -c '
      trap "exit 0" TERM INT
      while true; do
        dd if=/dev/urandom of=/tmp/blob bs=1M count=8 2>/dev/null
        wget -q -O /dev/null http://example.com/ 2>/dev/null || true
        echo "[$(date +%T)] flushed 8MiB and fetched upstream"
        rm -f /tmp/blob
      done'

  # --- a very long container name -----------------------------------------
  bold "a very long name (title truncation)"
  run this-is-an-extremely-long-container-name-that-should-truncate-in-the-title-bar \
    alpine:latest sh -c "$(noisy_cmd)"

  echo
  bold "what to look at"
  cat <<INSTRUCTIONS

  Run Dozzle with actions and shell enabled so the whole toolbar menu works:

    DOZZLE_ENABLE_ACTIONS=true DOZZLE_ENABLE_SHELL=true go run . --addr localhost:3100

  Then:

    payments-api            name dropdown, count badge, one stopped entry,
                            truncated image tag with copy on hover, cpu/mem limits
    ${PREFIX}-healthy       green health check
    ${PREFIX}-unhealthy     red health check
    ${PREFIX}-starting      neutral health check
    ${PREFIX}-linked        the open-in-new link icon
    ${PREFIX}-link-hint-*   the link discovery hint
    ${PREFIX}-volumes       volume dropdown (badge only if the disk is >=85% full)
    ${PREFIX}-busy          NET and DISK rows and live sparklines
    ${PREFIX}-this-is-an... long name truncation

  The link hint is dismissed permanently once you close it. To bring it back,
  clear DOZZLE_DISMISSEDLINKHINT from localStorage.

  Image update alerts are a separate fixture: ./scripts/fake-update.sh

INSTRUCTIONS
}

down() {
  require_docker

  bold "cleaning up"
  local ids
  ids=$(docker ps -aq --filter "name=^${PREFIX}-") || true
  if [ -n "$ids" ]; then
    # shellcheck disable=SC2086
    docker rm -f $ids >/dev/null
    green "  removed containers"
  else
    dim "  no containers"
  fi
  docker volume rm "$VOLUME" >/dev/null 2>&1 && green "  removed volume" || dim "  no volume"
  docker network rm "$NETWORK" >/dev/null 2>&1 && green "  removed network" || dim "  no network"
  docker rmi "$LONG_IMAGE" >/dev/null 2>&1 && green "  removed image" || dim "  no image"
  green "  done"
}

case "${1:-up}" in
  up) up ;;
  down) down ;;
  *)
    echo "usage: $0 [up|down]" >&2
    exit 1
    ;;
esac
