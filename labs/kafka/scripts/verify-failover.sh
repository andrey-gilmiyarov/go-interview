#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
lab_dir="$(cd "$script_dir/.." && pwd)"
compose_args=(--project-name kafka-handbook-cluster --file "$lab_dir/compose.cluster.yaml")
wait_timeout=${KAFKA_LAB_WAIT_TIMEOUT:-180}
[[ $wait_timeout =~ ^[0-9]+$ ]] && ((wait_timeout > 0)) || {
  printf 'KAFKA_LAB_WAIT_TIMEOUT must be a positive number of seconds\n' >&2
  exit 2
}

compose() {
  docker compose "${compose_args[@]}" "$@"
}

run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="kafka-handbook-failover-$run_id"
before_payload="before-$run_id"
after_payload="after-$run_id"
broker_stopped=0
topic_may_exist=0

cleanup() {
  local status=$?
  local topics=
  local deleted=0
  trap - EXIT INT TERM

  if [[ $broker_stopped == 1 ]]; then
    printf 'Restoring broker-1...\n'
    if ! compose up --detach --wait --wait-timeout "$wait_timeout" broker-1; then
      printf 'Could not confirm broker-1 is healthy after restart.\n' >&2
      status=1
    fi
  fi

  if [[ $topic_may_exist == 1 ]]; then
    if topics="$(compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-topics.sh \
      --bootstrap-server broker-2:19093 --list 2>/dev/null)"; then
      if printf '%s\n' "$topics" | grep --fixed-strings --line-regexp --quiet "$topic"; then
        printf 'Deleting test topic %s...\n' "$topic"
        if compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-topics.sh \
          --bootstrap-server broker-2:19093 --delete --topic "$topic" >/dev/null; then
          for ((attempt = 1; attempt <= 30; attempt++)); do
            if topics="$(compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-topics.sh \
              --bootstrap-server broker-2:19093 --list 2>/dev/null)" &&
              ! printf '%s\n' "$topics" | grep --fixed-strings --line-regexp --quiet "$topic"; then
              deleted=1
              break
            fi
            sleep 1
          done
        fi
        if [[ $deleted == 1 ]]; then
          printf 'Test topic removed.\n'
        else
          printf 'Could not confirm removal of test topic %s.\n' "$topic" >&2
          status=1
        fi
      fi
    else
      printf 'Could not list topics while cleaning up %s.\n' "$topic" >&2
      status=1
    fi
  fi

  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Ensuring the three-broker cluster is healthy...\n'
compose up --detach --wait --wait-timeout "$wait_timeout"
compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker-2:19093 describe --status

topic_may_exist=1
compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server broker-2:19093 --create --topic "$topic" \
  --partitions 1 --replication-factor 3 --config min.insync.replicas=2
topic_description="$(compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server broker-2:19093 --describe --topic "$topic")"
printf '%s\n' "$topic_description"
[[ $topic_description == *"ReplicationFactor: 3"* ]] || {
  printf 'Test topic did not get replication factor 3.\n' >&2
  exit 1
}
[[ $topic_description == *"min.insync.replicas=2"* ]] || {
  printf 'Test topic did not get min.insync.replicas=2.\n' >&2
  exit 1
}

produce_record() {
  local value=$1
  printf '%s\n' "$value" |
    compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-console-producer.sh \
      --bootstrap-server broker-2:19093 --topic "$topic" --sync \
      --command-property acks=all
}

printf 'Producing an acknowledged record before broker loss...\n'
produce_record "$before_payload"

broker_stopped=1
printf 'Stopping broker-1 and waiting for a different KRaft leader...\n'
compose stop --timeout 20 broker-1
leader_id=
quorum_status=
for ((attempt = 1; attempt <= 45; attempt++)); do
  if quorum_status="$(compose exec --no-TTY broker-2 \
    /opt/kafka/bin/kafka-metadata-quorum.sh --bootstrap-server broker-2:19093 \
    describe --status 2>/dev/null)"; then
    leader_id="$(printf '%s\n' "$quorum_status" | awk -F':[[:space:]]*' '$1 == "LeaderId" { print $2 }')"
    if [[ -n $leader_id && $leader_id != 1 ]]; then
      break
    fi
  fi
  sleep 1
done
if [[ -z $leader_id || $leader_id == 1 ]]; then
  printf 'No active KRaft leader was confirmed while broker-1 was stopped.\n' >&2
  [[ -z $quorum_status ]] || printf '%s\n' "$quorum_status" >&2
  exit 1
fi
printf '%s\n' "$quorum_status"

printf 'Producing an acknowledged record with broker-1 stopped...\n'
produce_record "$after_payload"

printf 'Reading both records back while broker-1 remains stopped...\n'
readback="$(compose exec --no-TTY broker-2 /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server broker-2:19093 --topic "$topic" --from-beginning \
  --max-messages 2 --timeout-ms 15000 \
  --formatter-property print.timestamp=false \
  --formatter-property print.key=false \
  --formatter-property print.offset=false \
  --formatter-property print.partition=false \
  --formatter-property print.value=true)"
expected="$(printf '%s\n%s' "$before_payload" "$after_payload")"
if [[ $readback != "$expected" ]]; then
  printf 'Expected records:\n%s\nActual records:\n%s\n' "$expected" "$readback" >&2
  exit 1
fi
printf '%s\n' "$readback"
printf 'RF=3/minISR=2 retained the earlier record and acknowledged/read the new record with broker-1 stopped.\n'
