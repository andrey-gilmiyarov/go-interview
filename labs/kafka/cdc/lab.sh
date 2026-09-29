#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage: lab.sh <kraft|zk> <up|ready|status|logs|down|reset>

  up      Start the selected isolated CDC environment, ensure its topic and
          connector configuration, then wait for a completed snapshot offset.
  ready   Wait until the connector task is running and its stored source offset
          contains an LSN outside the initial snapshot.
  status  Show containers, Connect/plugin versions, connector offsets, topic,
          PostgreSQL logical slot, and retained WAL.
  logs    Follow logs for the selected Compose project.
  down    Stop and remove the selected project's containers and network; keep
          named volumes.
  reset   Remove only the selected project's containers, network, and volumes.

Set CDC_LAB_WAIT_TIMEOUT to a positive number of seconds (default: 240).
USAGE
}

die() {
  printf 'lab.sh: %s\n' "$*" >&2
  exit 2
}

if (($# == 1)) && [[ $1 == -h || $1 == --help ]]; then
  usage
  exit 0
fi
if (($# != 2)); then
  usage >&2
  exit 2
fi

lab=$1
action=$2
case "$lab" in
  kraft)
    project=kafka-cdc-kraft
    compose_file=compose.kraft.yaml
    connect_url=http://127.0.0.1:18083
    kafka_bootstrap=broker:29092
    host_broker=127.0.0.1:49092
    host_database=127.0.0.1:25432
    ;;
  zk)
    project=kafka-cdc-zk
    compose_file=compose.zk.yaml
    connect_url=http://127.0.0.1:28083
    kafka_bootstrap=broker:19093
    host_broker=127.0.0.1:59092
    host_database=127.0.0.1:35432
    ;;
  *)
    die "unknown CDC environment: $lab (expected kraft or zk)"
    ;;
esac

case "$action" in
  up|ready|status|logs|down|reset) ;;
  *) die "unknown action: $action" ;;
esac

wait_timeout=${CDC_LAB_WAIT_TIMEOUT:-240}
[[ $wait_timeout =~ ^[0-9]+$ ]] && ((wait_timeout > 0 && wait_timeout <= 1800)) || die 'CDC_LAB_WAIT_TIMEOUT must be an integer from 1 through 1800 seconds'

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
compose_path="$script_dir/$compose_file"
connector_path="$script_dir/connector.json"
connector_name=cdc-orders

dc() {
  docker compose --project-name "$project" --file "$compose_path" "$@"
}

rest() {
  curl --fail --silent --show-error --connect-timeout 2 --max-time 5 "$@"
}

wait_for_connect() {
  local deadline=$((SECONDS + wait_timeout))
  local response
  while ((SECONDS < deadline)); do
    if response=$(rest "$connect_url/"); then
      printf 'Kafka Connect REST is available at %s\n' "$connect_url"
      return 0
    fi
    sleep 1
  done
  die "Kafka Connect REST did not become available within ${wait_timeout}s"
}

ensure_topic() {
  dc exec -T broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --create --if-not-exists --topic cdc.orders \
    --partitions 3 --replication-factor 1 \
    --config cleanup.policy=delete \
    --config retention.ms=604800000 >/dev/null

  local description
  description=$(dc exec -T broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --describe --topic cdc.orders)
  if ! python3 -c '
import re, sys
first = sys.stdin.read().splitlines()[0]
match = re.search(r"PartitionCount:\s*(\d+)\s+ReplicationFactor:\s*(\d+)\s+Configs:\s*(.*)$", first)
if not match:
    print(f"cannot parse cdc.orders description: {first}", file=sys.stderr)
    raise SystemExit(1)
partitions, replicas = map(int, match.group(1, 2))
configs = dict(item.split("=", 1) for item in match.group(3).split(",") if "=" in item)
required = {"cleanup.policy": "delete", "retention.ms": "604800000"}
if partitions != 3 or replicas != 1 or any(configs.get(k) != v for k, v in required.items()):
    print(f"cdc.orders has incompatible settings: {first}", file=sys.stderr)
    raise SystemExit(1)
' <<<"$description"; then
    die 'cdc.orders exists with a different partition count, replication factor, or retention configuration; refusing to alter it'
  fi
  printf 'Topic cdc.orders is present with 3 partitions, RF=1, and seven-day delete retention\n'
}

ensure_heartbeat_topic() {
  local heartbeat_topic=__debezium-heartbeat.cdc
  dc exec -T broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --create --if-not-exists --topic "$heartbeat_topic" \
    --partitions 1 --replication-factor 1 \
    --config cleanup.policy=compact >/dev/null

  local description
  description=$(dc exec -T broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --describe --topic "$heartbeat_topic")
  if ! python3 -c '
import re, sys
first = sys.stdin.read().splitlines()[0]
match = re.search(r"PartitionCount:\s*(\d+)\s+ReplicationFactor:\s*(\d+)\s+Configs:\s*(.*)$", first)
if not match:
    print(f"cannot parse heartbeat topic description: {first}", file=sys.stderr)
    raise SystemExit(1)
partitions, replicas = map(int, match.group(1, 2))
configs = dict(item.split("=", 1) for item in match.group(3).split(",") if "=" in item)
if partitions != 1 or replicas != 1 or configs.get("cleanup.policy") != "compact":
    print(f"heartbeat topic has incompatible settings: {first}", file=sys.stderr)
    raise SystemExit(1)
' <<<"$description"; then
    die "$heartbeat_topic exists with incompatible partition, replication, or compaction settings; refusing to alter it"
  fi
  printf 'Topic %s is present with 1 partition, RF=1, and compaction enabled\n' "$heartbeat_topic"
}

ensure_connector() {
  local connector_names expected current
  connector_names=$(rest "$connect_url/connectors") || die 'cannot list Kafka Connect connectors'
  if python3 -c 'import json,sys; raise SystemExit(0 if sys.argv[1] in json.load(sys.stdin) else 1)' "$connector_name" <<<"$connector_names"; then
    current=$(rest "$connect_url/connectors/$connector_name/config") || die "cannot read existing connector $connector_name configuration"
    expected=$(python3 -c 'import json,sys; print(json.dumps(json.load(open(sys.argv[1]))["config"], sort_keys=True, separators=(",", ":")))' "$connector_path")
    if ! python3 -c '
import json, sys
expected = json.loads(sys.argv[1])
actual = json.loads(sys.argv[2])
if actual.get("name") == "cdc-orders":
    actual = {key: value for key, value in actual.items() if key != "name"}
if expected != actual:
    differing = sorted(k for k in set(expected) | set(actual) if expected.get(k) != actual.get(k))
    print("existing connector configuration differs at: " + ", ".join(differing), file=sys.stderr)
    raise SystemExit(1)
' "$expected" "$current"; then
      die "existing connector $connector_name differs from connector.json; refusing to overwrite it"
    fi
    printf 'Connector %s already has the approved configuration; leaving it unchanged\n' "$connector_name"
    return
  fi

  rest -X POST -H 'Content-Type: application/json' --data-binary "@$connector_path" "$connect_url/connectors" >/dev/null \
    || die "cannot create connector $connector_name"
  printf 'Created connector %s from connector.json\n' "$connector_name"
}

ready_once() {
  local status offsets
  status=$(rest "$connect_url/connectors/$connector_name/status" 2>/dev/null) || return 1
  offsets=$(rest "$connect_url/connectors/$connector_name/offsets" 2>/dev/null) || return 1
  python3 -c '
import json, sys
status = json.loads(sys.argv[1])
offsets = json.loads(sys.argv[2])
if status.get("connector", {}).get("state") != "RUNNING":
    raise SystemExit(1)
tasks = status.get("tasks", [])
if not tasks or any(task.get("state") != "RUNNING" for task in tasks):
    raise SystemExit(1)
for item in offsets.get("offsets", []):
    offset = item.get("offset") or {}
    lsn = offset.get("lsn")
    if lsn is None or str(lsn).strip() in ("", "0", "0/0"):
        continue
    snapshot = offset.get("snapshot")
    # `last` marks the final snapshot record; it is complete once its offset is stored.
    if snapshot is True or str(snapshot).lower() == "true":
        continue
    if snapshot not in (None, False, "false", "last", "completed"):
        continue
    print(lsn)
    raise SystemExit(0)
raise SystemExit(1)
' "$status" "$offsets"
}

wait_until_ready() {
  local deadline=$((SECONDS + wait_timeout))
  local lsn
  while ((SECONDS < deadline)); do
    if lsn=$(ready_once); then
      printf 'Ready: connector %s is RUNNING; snapshot completion is recorded with source LSN %s\n' "$connector_name" "$lsn"
      return 0
    fi
    sleep 1
  done
  printf 'Last connector status:\n' >&2
  rest "$connect_url/connectors/$connector_name/status" >&2 || true
  printf '\nLast source offsets:\n' >&2
  rest "$connect_url/connectors/$connector_name/offsets" >&2 || true
  die "connector snapshot and persisted LSN did not become ready within ${wait_timeout}s"
}

show_status() {
  dc ps --all
  printf '\nKafka Connect worker version:\n'
  rest "$connect_url/" | python3 -c 'import json,sys; x=json.load(sys.stdin); print("version={version} commit={commit} cluster_id={kafka_cluster_id}".format(**x))' || true
  printf '\nDebezium PostgreSQL plugin version:\n'
  rest "$connect_url/connector-plugins" | python3 -c '
import json,sys
for plugin in json.load(sys.stdin):
    if plugin.get("class") == "io.debezium.connector.postgresql.PostgresConnector":
        print(plugin)
' || true
  printf '\nConnector status:\n'
  rest "$connect_url/connectors/$connector_name/status" || true
  printf '\nConnector offsets:\n'
  rest "$connect_url/connectors/$connector_name/offsets" || true
  printf '\nCDC topic:\n'
  dc exec -T broker /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --describe --topic cdc.orders || true
  printf '\nKafka consumer group lag:\n'
  dc exec -T broker /opt/kafka/bin/kafka-consumer-groups.sh \
    --bootstrap-server "$kafka_bootstrap" \
    --command-config /mnt/shared/config/admin.properties \
    --all-groups --describe || true
  printf '\nPostgreSQL logical slot and retained WAL:\n'
  dc exec -T -e PGCONNECT_TIMEOUT=3 -e 'PGOPTIONS=-c statement_timeout=5000 -c lock_timeout=1000' postgres psql -U cdc_admin -d cdc_lab -c \
    "SELECT slot_name, plugin, active, restart_lsn, confirmed_flush_lsn, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)::bigint) AS retained_wal FROM pg_replication_slots;" || true
  printf '\nPostgreSQL CDC settings:\n'
  dc exec -T -e PGCONNECT_TIMEOUT=3 -e 'PGOPTIONS=-c statement_timeout=5000 -c lock_timeout=1000' postgres psql -U cdc_admin -d cdc_lab -c \
    "SHOW wal_level; SHOW max_replication_slots; SHOW max_wal_senders; SHOW max_slot_wal_keep_size;" || true
}

case "$action" in
  up)
    dc up --detach --wait --wait-timeout "$wait_timeout"
    wait_for_connect
    ensure_topic
    ensure_heartbeat_topic
    ensure_connector
    wait_until_ready
    printf 'Host endpoints: broker=%s database=%s connect=%s\n' "$host_broker" "$host_database" "$connect_url"
    ;;
  ready)
    wait_for_connect
    wait_until_ready
    ;;
  status)
    show_status
    ;;
  logs)
    dc logs --tail 100 --follow
    ;;
  down)
    dc down --remove-orphans
    ;;
  reset)
    printf 'Removing only CDC project %s and its named volumes.\n' "$project"
    dc down --remove-orphans --volumes
    ;;
esac
