#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage: verify.sh <kraft|zk>

Checks readiness for one CDC environment, then runs the bounded CDC end-to-end
tests against its PostgreSQL, Kafka, and Kafka Connect services.
USAGE
}

die() {
  printf 'verify.sh: %s\n' "$*" >&2
  exit 2
}

if (($# == 1)) && [[ $1 == -h || $1 == --help ]]; then
  usage
  exit 0
fi
if (($# != 1)); then
  usage >&2
  exit 2
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
module_dir="$(cd -- "$script_dir/.." && pwd)"
mode=$1

case "$mode" in
  kraft)
    project=kafka-cdc-kraft
    database_port=25432
    broker=127.0.0.1:49092
    connect_url=http://127.0.0.1:18083
    ;;
  zk)
    project=kafka-cdc-zk
    database_port=35432
    broker=127.0.0.1:59092
    connect_url=http://127.0.0.1:28083
    ;;
  *) die "unknown CDC environment: $mode (expected kraft or zk)" ;;
esac

export CDC_MODE="$mode"
export CDC_COMPOSE_PROJECT="$project"
export CDC_COMPOSE_FILE="$script_dir/compose.$mode.yaml"
export CDC_DATABASE_URL="postgres://cdc_app:cdc_app@127.0.0.1:${database_port}/cdc_lab?sslmode=disable"
export CDC_ADMIN_DATABASE_URL="postgres://cdc_admin:cdc_admin@127.0.0.1:${database_port}/cdc_lab?sslmode=disable"
export CDC_KAFKA_BROKERS="$broker"
export CDC_CONNECT_URL="$connect_url"
export CDC_TOPIC=cdc.orders
export CDC_LAB_WAIT_TIMEOUT="${CDC_LAB_WAIT_TIMEOUT:-120}"

[[ $CDC_LAB_WAIT_TIMEOUT =~ ^[0-9]+$ ]] && ((CDC_LAB_WAIT_TIMEOUT > 0 && CDC_LAB_WAIT_TIMEOUT <= 120)) \
  || die 'CDC_LAB_WAIT_TIMEOUT must be an integer from 1 through 120 seconds'

cd "$module_dir"
bash cdc/lab.sh "$mode" ready

restore_selected_services_on_failure() {
  local exit_code=$?
  trap - EXIT
  if ((exit_code == 0)); then
    exit 0
  fi

  printf 'verify.sh: restoring broker and Connect for CDC project %s after test failure\n' "$project" >&2
  if ! docker compose --project-name "$project" --file "$CDC_COMPOSE_FILE" start broker connect; then
    printf 'verify.sh: could not start broker and Connect for %s\n' "$project" >&2
  elif ! CDC_LAB_WAIT_TIMEOUT=60 bash cdc/lab.sh "$mode" ready; then
    printf 'verify.sh: selected CDC services started but did not return to ready state\n' >&2
  fi
  exit "$exit_code"
}

trap restore_selected_services_on_failure EXIT
go test -tags cdcintegration -count=1 -timeout=8m ./internal/cdce2e
