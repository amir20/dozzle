#!/usr/bin/env bash
# Starts a sandboxed Dozzle that only sees five fictional busybox containers, so a
# capture never shows a real container name. `./demo.sh down` removes all of it.
#
# Each container runs an image named after a real app, so it gets that app's icon.
# Most are busybox re-tagged. Two (`tag=older`) are a real older release tagged as the
# moving tag, so Dozzle's update check finds a genuinely newer image for them.
# A tag is only created when it doesn't exist yet, and only removed if it still points
# at what this script put there, so a real local image is never touched.
#
# DOZZLE_IMAGE picks the Dozzle build (default amir20/dozzle:latest; a PR image such as
# amir20/dozzle:pr-1234 shows unreleased UI).
set -euo pipefail
cd "$(dirname "$0")"
PORT=${PORT:-3190}
DOZZLE_IMAGE=${DOZZLE_IMAGE:-amir20/dozzle:latest}
BASE=busybox:1.36
# name:generator:image[=older image to tag as it]
CONTAINERS=(
  checkout-api:api:node:22.9-alpine
  orders-worker:worker:python:3.12.6-alpine
  web:web:nginx:1.27-alpine=nginx:1.27.0-alpine
  postgres:db:postgres:16.4-alpine
  redis:redis:redis:7.2-alpine=redis:7.2.4-alpine
)

down() {
  local c
  for c in "${CONTAINERS[@]}"; do docker rm -f "${c%%:*}" >/dev/null 2>&1 || true; done
  docker rm -f tour-dozzle >/dev/null 2>&1 || true

  for c in "${CONTAINERS[@]}"; do
    local spec=${c#*:*:} img from id
    img=${spec%%=*}
    from=$BASE
    [[ "$spec" == *=* ]] && from=${spec#*=}
    id=$(docker image inspect -f '{{.Id}}' "$img" 2>/dev/null || true)
    [[ -n "$id" && "$id" == "$(docker image inspect -f '{{.Id}}' "$from" 2>/dev/null)" ]] && docker rmi "$img" >/dev/null || true
    [[ "$from" != "$BASE" ]] && { docker rmi "$from" >/dev/null 2>&1 || true; }
  done
  return 0
}

if [[ "${1:-up}" == "down" ]]; then
  down
  exit 0
fi

down
docker pull -q "$BASE" >/dev/null
mkdir -p work/data
for c in "${CONTAINERS[@]}"; do
  name=${c%%:*}
  rest=${c#*:}
  gen=${rest%%:*}
  spec=${rest#*:}
  img=${spec%%=*}
  from=$BASE
  if [[ "$spec" == *=* ]]; then
    from=${spec#*=}
    docker pull -q "$from" >/dev/null
  fi
  docker image inspect "$img" >/dev/null 2>&1 || docker tag "$from" "$img"
  docker run -d --name "$name" --label tour-video=1 --label dev.dozzle.group=shop \
    -v "$PWD/gen/$gen.sh:/run.sh:ro" "$img" sh /run.sh >/dev/null
done
docker run -d --name tour-dozzle -p "$PORT:8080" \
  -v /var/run/docker.sock:/var/run/docker.sock -v "$PWD/work/data:/data" \
  -e DOZZLE_FILTER=label=tour-video=1 -e DOZZLE_HOSTNAME=prod-1 -e DOZZLE_NO_ANALYTICS=1 -e DOZZLE_ENABLE_ACTIONS=true \
  "$DOZZLE_IMAGE" >/dev/null
echo "Dozzle ($DOZZLE_IMAGE) on http://localhost:$PORT. Let it run ~3 minutes so the SQL query has rows."
