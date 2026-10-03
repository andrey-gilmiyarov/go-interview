#!/usr/bin/env bash
set -Eeuo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
export POSTGRES_LAB_PORT=${POSTGRES_LAB_PORT:-45432}
[[ $POSTGRES_LAB_PORT =~ ^[0-9]+$ ]] && ((POSTGRES_LAB_PORT > 1024 && POSTGRES_LAB_PORT < 65536)) || { echo 'POSTGRES_LAB_PORT must be 1025..65535' >&2; exit 2; }
dc() { docker compose --project-name postgres-handbook --file compose.yaml "$@"; }
ready() { dc exec -T postgres pg_isready -U postgres_lab -d postgres_lab >/dev/null; }
case "${1:-}" in
 up) dc up -d --wait --wait-timeout 90; ready ;;
 ready) ready ;;
 status) dc ps ;;
 logs) dc logs --tail=100 postgres ;;
 down) dc down ;;
 reset) dc down --volumes ;;
 psql) dc exec postgres psql -U postgres_lab -d postgres_lab ;;
 run) [[ $# == 2 ]] || { echo 'Usage: ./lab.sh run <scenario-id>' >&2; exit 2; }; go run ./cmd/lab "$2" ;;
 *) echo 'Usage: ./lab.sh up|ready|status|logs|down|reset|psql|run <scenario-id>' >&2; exit 2 ;;
esac
