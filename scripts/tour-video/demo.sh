#!/usr/bin/env bash
# Starts a sandboxed Dozzle that only sees five fictional busybox containers, so a
# capture never shows a real container name. `./demo.sh down` removes all of it.
set -euo pipefail
cd "$(dirname "$0")"
PORT=${PORT:-3190}
NAMES=(checkout-api:api orders-worker:worker web:web postgres:db redis:redis)

down() {
  docker rm -f tour-dozzle "${NAMES[@]%%:*}" >/dev/null 2>&1 || true
}

if [[ "${1:-up}" == "down" ]]; then
  down
  exit 0
fi

down
mkdir -p work/data
for n in "${NAMES[@]}"; do
  docker run -d --name "${n%%:*}" --label tour-video=1 --label dev.dozzle.group=shop \
    -v "$PWD/gen/${n##*:}.sh:/run.sh:ro" busybox:1.36 sh /run.sh >/dev/null
done
docker run -d --name tour-dozzle -p "$PORT:8080" \
  -v /var/run/docker.sock:/var/run/docker.sock -v "$PWD/work/data:/data" \
  -e DOZZLE_FILTER=label=tour-video=1 -e DOZZLE_HOSTNAME=prod-1 -e DOZZLE_NO_ANALYTICS=1 \
  amir20/dozzle:latest >/dev/null
echo "Dozzle on http://localhost:$PORT. Let it run ~3 minutes so the SQL query has rows."
