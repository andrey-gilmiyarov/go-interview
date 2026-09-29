#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage: lab.sh <kraft|zk|cluster> [up|down|reset|status|logs] [--outbox]

  up       Start services and wait up to KAFKA_LAB_WAIT_TIMEOUT seconds (default 180).
           Use --outbox with kraft to start optional PostgreSQL.
  down     Stop and remove this lab's containers and network, keeping volumes.
  reset    Remove this lab's containers, network, and named volumes.
  status   Show containers for this project.
  logs     Follow this project's logs.

Examples:
  ./labs/kafka/scripts/lab.sh kraft up --outbox
  ./labs/kafka/scripts/lab.sh cluster status
  ./labs/kafka/scripts/lab.sh zk reset
USAGE
}

die() {
  printf 'lab.sh: %s\n' "$*" >&2
  exit 2
}

if (($# == 0)); then
  usage >&2
  exit 2
fi

lab=$1
shift
action=up
if (($# > 0)) && [[ $1 != --* ]]; then
  action=$1
  shift
fi

outbox=0
while (($# > 0)); do
  case $1 in
    --outbox)
      [[ $outbox == 0 ]] || die '--outbox was supplied more than once'
      outbox=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

case $lab in
  kraft)
    project=kafka-handbook-kraft
    compose_file=compose.kraft.yaml
    ;;
  zk)
    project=kafka-handbook-zk
    compose_file=compose.zk.yaml
    ;;
  cluster)
    project=kafka-handbook-cluster
    compose_file=compose.cluster.yaml
    ;;
  *)
    die "unknown lab: $lab"
    ;;
esac

if [[ $outbox == 1 && $lab != kraft ]]; then
  die '--outbox is available only with the kraft lab'
fi

case $action in
  up|down|reset|status|logs) ;;
  *)
    die "unknown action: $action"
    ;;
esac

wait_timeout=${KAFKA_LAB_WAIT_TIMEOUT:-180}
[[ $wait_timeout =~ ^[0-9]+$ ]] && ((wait_timeout > 0)) || die 'KAFKA_LAB_WAIT_TIMEOUT must be a positive number of seconds'

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
lab_dir="$(cd "$script_dir/.." && pwd)"
compose_args=(--project-name "$project" --file "$lab_dir/$compose_file")

# Add the profile for a requested up, and for commands that must see every
# service in this project's declared model (including an already-running outbox).
if [[ $lab == kraft && ( $outbox == 1 || $action != up ) ]]; then
  compose_args+=(--profile outbox)
fi

dc() {
  docker compose "${compose_args[@]}" "$@"
}

case $action in
  up)
    dc up --detach --wait --wait-timeout "$wait_timeout"
    ;;
  down)
    dc down --remove-orphans
    ;;
  reset)
    printf 'Removing containers, network, and named volumes for %s only.\n' "$project"
    dc down --remove-orphans --volumes
    ;;
  status)
    dc ps --all
    ;;
  logs)
    dc logs --tail 100 --follow
    ;;
esac
