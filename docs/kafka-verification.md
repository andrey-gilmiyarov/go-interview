# Приёмка Kafka handbook и Go-лаборатории

Дата: 2026-09-29. Рабочее дерево: kafka-handbook.

## Результат

Обзор, восемь глав, реестр и лаборатория приняты Lead после технического review и исправлений. Код использует franz-go; outbox — PostgreSQL + polling. CDC/Debezium остаётся объяснением и отдельным пунктом roadmap. Корневые Go-зависимости не изменены. Коммитов и staging не выполнялось.

## Основные проверки

Команды ниже выполнены в корне рабочего дерева, если не указано иначе; PASS означает exit 0.

| Команда | Результат |
| --- | --- |
| `npm run test:site` | PASS, 37/37 |
| `npm run docs:check` | PASS: 8 групп, 42 темы Go, 20 упражнений, 8 тем Kafka |
| `npm run docs:build` | PASS; повторено после окончательного исправления producer, 3.55 s |
| `go test ./...` | PASS, корневой модуль |
| `go build ./...` | PASS, корневой модуль |
| `go test ./...` в `labs/kafka` | PASS |
| `go test -race ./...` в `labs/kafka` | PASS |
| `go build ./...` в `labs/kafka` | PASS |
| `go vet ./...` в `labs/kafka` | PASS |

Полные Go-прогоны предшествуют двум последним локальным исправлениям review: приоритету уже полученного callback при отмене producer context и ограничению времени очистки тестовой схемы. После них выполнены целевые проверки, перечисленные ниже. Сборка сайта предупреждает о chunks больше 500 kB; сборка завершилась успешно.

## Проверка браузером

Собранный сайт проверен через локальный preview: верхняя навигация, порядок sidebar, поиск Kafka, раскрытие скрытых ответов, переход к outbox из мобильного меню. На ширине 390 px горизонтального переполнения нет. Лаборатория отображает README как Markdown с одним H1. После последней сборки обзор повторно открыт. Preview: http://127.0.0.1:4173/kafka/.

## Ограничения проверки

Локальные контейнеры выполнялись на arm64; наличие amd64-образов проверено по manifest, запуск на amd64 не выполнялся. Отказ брокера проверен отдельным shell-сценарием RF3/minISR2; opt-in Go-тест RF3 в обычных интеграционных прогонах пропущен. Это учебная лаборатория, не проверка production-нагрузки.

## Подробные команды и результаты исполнителей

Ниже исторические отчёты этапов; упоминания запущенных контейнеров относятся к моменту соответствующего отчёта. Итоговое состояние приведено в конце.


---

# Kafka lab infrastructure execution

Worktree: `/Users/andreigilmiiarov/.codex/worktrees/kafka-handbook/go-interview`

## Files

- `labs/kafka/compose.kraft.yaml` — one Kafka 4.3.1 KRaft broker and optional PostgreSQL 17.6 in the `outbox` profile.
- `labs/kafka/compose.zk.yaml` plus `labs/kafka/docker/kafka-zk/server.properties` — ZooKeeper 3.9.3 and Kafka 3.9.2 started in ZooKeeper mode.
- `labs/kafka/compose.cluster.yaml` — three combined-role Kafka 4.3.1 KRaft brokers, RF=3/minISR=2 defaults.
- `labs/kafka/scripts/lab.sh` — project-scoped `up|down|reset|status|logs`; Compose files resolve relative to the script.
- `labs/kafka/scripts/verify-failover.sh` — create/write/read/cleanup failover check with broker restore via EXIT trap.

Use `./labs/kafka/scripts/lab.sh kraft up --outbox` for the Go integration environment. Host endpoints: KRaft `127.0.0.1:19092`; ZooKeeper Kafka `127.0.0.1:29092`; cluster brokers `127.0.0.1:39092`, `:39093`, `:39094`; PostgreSQL `127.0.0.1:15432` (`kafka_lab/kafka_lab`).

## Images

All pinned tags pulled successfully:

- `docker pull apache/kafka:4.3.1` — PASS; manifest digest `sha256:77e3df9054047a88b520d0cc46e16696d3b22022e1d580aeccd2632df6532837`.
- `docker pull apache/kafka:3.9.2` — PASS; manifest digest `sha256:05b4616e0702ef2729327705d54ad6b50ea70b271c4b730fabd2320789fb7b02`.
- `docker pull zookeeper:3.9.3` — PASS; manifest digest `sha256:9980cafbff742c15b339811ae829faa61c69154606ec504223560da9d31acd43`.
- `docker pull postgres:17.6` — PASS; manifest digest `sha256:00bc86618629af00d2937fdc5a5d63db3ff8450acf52f0636ec813c7f4902929`.

Runtime version commands passed: KRaft broker reported Kafka `4.3.1`; ZooKeeper-mode broker reported Kafka `3.9.2`; ZooKeeper reported `3.9.3`; `psql --version` reported PostgreSQL `17.6` (`17.6-2.pgdg13+1`). The locally selected platform is linux/arm64 for each image.

For each image below, `docker buildx imagetools inspect --raw <image> | jq -r '[.manifests[].platform | select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64")) | .os + "/" + .architecture] | unique | .[]'` returned both listed platforms:

| Image | Platforms |
| --- | --- |
| `apache/kafka:4.3.1` | linux/amd64, linux/arm64 |
| `apache/kafka:3.9.2` | linux/amd64, linux/arm64 |
| `zookeeper:3.9.3` | linux/amd64, linux/arm64 |
| `postgres:17.6` | linux/amd64, linux/arm64 |

## Validation commands and results

- `docker compose --project-name kafka-handbook-kraft --file labs/kafka/compose.kraft.yaml config --quiet` — PASS.
- `docker compose --project-name kafka-handbook-zk --file labs/kafka/compose.zk.yaml config --quiet` — PASS.
- `docker compose --project-name kafka-handbook-cluster --file labs/kafka/compose.cluster.yaml config --quiet` — PASS.
- `bash -n labs/kafka/scripts/lab.sh labs/kafka/scripts/verify-failover.sh` — PASS.
- `./labs/kafka/scripts/lab.sh kraft up --outbox` — PASS; broker and PostgreSQL became healthy.
- `./labs/kafka/scripts/lab.sh zk up` — PASS; ZooKeeper health dependency and Kafka broker became healthy.
- `./labs/kafka/scripts/lab.sh cluster up` — PASS; all three brokers became healthy.
- `./labs/kafka/scripts/verify-failover.sh` — PASS. It created a unique topic described as RF=3/minISR=2 with replicas/ISR 1,2,3; produced an `acks=all` record; polled until a new KRaft leader was active after stopping broker-1; produced a second `acks=all` record; and consumed both expected payloads while broker-1 was stopped. The EXIT trap restored broker-1 and confirmed the test topic was removed.
- `./labs/kafka/scripts/lab.sh cluster down` — PASS; removed only cluster containers/network and preserved `kafka-handbook-cluster_broker_1_data`, `_broker_2_data`, and `_broker_3_data`.
- `./labs/kafka/scripts/lab.sh kraft status` — PASS; broker and outbox PostgreSQL healthy.
- `./labs/kafka/scripts/lab.sh zk status` — PASS; broker and ZooKeeper healthy.
- `./labs/kafka/scripts/lab.sh cluster status` — PASS; no cluster containers remain.
- Absolute-path `lab.sh kraft status` with working directory `/tmp` — PASS; confirms the script is independent of the caller's current directory.
- `git diff --check` — PASS.
- `git diff --cached --check` — PASS.
- `if rg -n '[[:blank:]]+$' labs/kafka; then exit 1; else exit 0; fi` — PASS; no trailing whitespace in assigned files.

Final state: `kafka-handbook-kraft` with outbox and `kafka-handbook-zk` remain healthy for Go integration. `kafka-handbook-cluster` is down and its named volumes are preserved. No unrelated Compose projects were started, stopped, or removed.

## Exact version and manifest probes

- `docker compose --project-name kafka-handbook-kraft --file labs/kafka/compose.kraft.yaml exec -T broker /opt/kafka/bin/kafka-topics.sh --version` — PASS, `4.3.1`.
- `docker compose --project-name kafka-handbook-zk --file labs/kafka/compose.zk.yaml exec -T broker /opt/kafka/bin/kafka-topics.sh --version` — PASS, `3.9.2`.
- `docker compose --project-name kafka-handbook-zk --file labs/kafka/compose.zk.yaml exec -T zookeeper zkServer.sh version` — PASS, `3.9.3`.
- `docker compose --project-name kafka-handbook-kraft --file labs/kafka/compose.kraft.yaml exec -T outbox-postgres sh -c 'psql --version && pg_isready -U kafka_lab -d kafka_lab'` — PASS, PostgreSQL client `17.6`; server accepts connections.
- `docker buildx imagetools inspect --raw apache/kafka:4.3.1 | jq -r '[.manifests[].platform | select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64")) | .os + "/" + .architecture] | unique | .[]'` — PASS, linux/amd64 and linux/arm64.
- `docker buildx imagetools inspect --raw apache/kafka:3.9.2 | jq -r '[.manifests[].platform | select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64")) | .os + "/" + .architecture] | unique | .[]'` — PASS, linux/amd64 and linux/arm64.
- `docker buildx imagetools inspect --raw zookeeper:3.9.3 | jq -r '[.manifests[].platform | select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64")) | .os + "/" + .architecture] | unique | .[]'` — PASS, linux/amd64 and linux/arm64.
- `docker buildx imagetools inspect --raw postgres:17.6 | jq -r '[.manifests[].platform | select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64")) | .os + "/" + .architecture] | unique | .[]'` — PASS, linux/amd64 and linux/arm64.


---

# Go Kafka lab worker report

Task: `/root/kafka_go`
Status: READY_FOR_REVIEW

## Changed

- `labs/kafka/go.mod`, `go.sum`
- `labs/kafka/internal/config/config.go`, `config_test.go`
- `labs/kafka/internal/event/event.go`, `event_test.go`
- `labs/kafka/internal/kafka/client.go`, `client_test.go`, `consume.go`
- `labs/kafka/internal/integration/kafka_test.go`

The store, outbox, and CLI command files were implemented by their assigned workers and were not edited here. No stage or commit was made.

## Implementation

- Added pinned franz-go 1.22.1 and pgx/v5 5.11.0 module dependencies with the expected transitive graph, Go 1.27.0 and toolchain 1.27.1.
- Added validated environment config and versioned JSON order events.
- Added Kafka producer/consumer constructors, raw kmsg topic and group cleanup, event encode/decode, and async confirmed production. `ProduceEvent` now prefers an already-buffered callback result if cancellation wins a simultaneous select; only a missing callback result is reported as `ErrDeliveryOutcomeUnknown`.
- Added sequential manual-commit processing, ordered one-worker-per-partition processing with bounded 64-record polls, 10-second processing/commit context, cancellation and join before commit, and rebalance release on every poll path.
- Added transactional Kafka-to-Kafka processing with `ReadCommitted`, bounded callback waits, a fresh 5-second abort context, output/offset atomicity, and retry when `End(..., TryCommit)` returns `committed=false`.
- Added live tests for sequential redelivery, later-offset safety after an earlier same-partition failure, a two-consumer blocked rebalance, transaction abort invisibility and persisted group-offset recovery, negotiated topic deletion, and an opt-in RF3/minISR topic case. The RF3 case runs only when `KAFKA_REPLICATION_FACTOR=3` is selected.

## TDD evidence

- RED — `go test ./internal/event ./internal/config`: failed before implementation with missing `Load`, `Encode`, `Event`, and `Decode` APIs; GREEN — the same packages passed after implementation.
- RED — `go test -tags=integration ./internal/integration -run '^TestSequentialProcessingFailureBeforeCommitRedelivers$' -count=1`: failed to compile because runner APIs and failpoint sentinels were not yet implemented; GREEN — the sequential redelivery integration passed after the runners landed.
- RED — `go test ./internal/kafka -run 'TestDeliveryCallbackAlreadyReadyWinsOverCanceledContext|TestDeliveryWithoutReadyCallbackKeepsUnknownOutcome' -count=1`: failed to compile because the delivery wait helper/type did not yet exist; GREEN — the same command passed after the callback-drain fix.

## Validation

- `go test -count=1 ./internal/kafka` — PASS after the final callback-race fix.
- `KAFKA_BROKERS=127.0.0.1:19092 go test -tags=integration -count=1 ./internal/integration` — PASS after the final callback-race fix, including Kafka 4.3.1 and the optional RF3 test compile/skip under the default RF1 setting.
- `KAFKA_BROKERS=127.0.0.1:19092 go test -tags=integration ./... -count=1` — PASS before the final callback-race fix.
- `KAFKA_BROKERS=127.0.0.1:29092 go test -tags=integration ./... -count=1` — PASS before the final callback-race fix.
- `KAFKA_BROKERS=127.0.0.1:19092 go test -race -tags=integration -count=1 ./...` — PASS before the final callback-race fix.
- `go test ./...` — PASS before the final callback-race fix.
- `go test -race ./...` — PASS before the final callback-race fix.
- `go build ./...` — PASS before the final callback-race fix.
- `go test -tags=integration -run '^$' ./internal/integration` — PASS; compiled the final tagged test package without running tests.
- `gofmt -l internal/config/config.go internal/config/config_test.go internal/event/event.go internal/event/event_test.go internal/integration/kafka_test.go internal/kafka/client.go internal/kafka/client_test.go internal/kafka/consume.go` — PASS; no output.
- `git diff --check` — PASS for tracked changes. The new `labs/kafka` tree is untracked in the shared checkout, so this command does not inspect its contents; Go formatting and the test/build commands cover those files.

The RF3/minISR runtime scenario was not run; no three-broker endpoint was confirmed during this task. The test is opt-in and will fail on topic creation if explicitly selected against an unavailable or undersized cluster.


---

# Kafka CLI wrapper report

Task: T3c wrappers. Changed only `labs/kafka/cmd/{producer,consumer,parallel-consumer,transactions}/`.

## Implementation

- Added host-side CLIs with the approved flags. All commands default to a 60-second overall timeout, use SIGINT/SIGTERM cancellation, emit JSON `slog` records to stdout, and return failures on stderr with a nonzero process status.
- Producer defaults to the configured topic, one event, and 12,500 cents. It generates unique event IDs and, unless `-order-id` is supplied, a distinct order ID for each event. A fixed `-event-id` is accepted only for a single event.
- Consumer commands default to configured topic/group, create three-partition topics with RF from config and min ISR 1 for RF=1 or 2 otherwise, and use read-committed isolation. Parallel consumer enables `BlockRebalanceOnPoll` and closes with rebalance release.
- Transaction command defaults output topic to `<input>-processed` and transactional ID to `<group>-transactions`; flag help explains that simultaneous instances need unique IDs. Input/output equality is rejected. Consumer and transaction shutdown attempt `LeaveGroupContext` with a fresh five-second context before `CloseAllowingRebalance`.
- Wrappers call the shared Kafka helpers and runners; business processing/commit behavior remains in `internal/kafka`.

## Validation

- `go test ./cmd/...` — PASS.
- `go test ./...` — PASS.
- `go build ./cmd/...` — PASS.
- `go build ./...` — PASS.
- `go vet ./cmd/producer ./cmd/consumer ./cmd/parallel-consumer ./cmd/transactions` — PASS.
- `gofmt -d` on all four `main.go` files and producer test — no output.
- `rg -n '[[:blank:]]+$'` across the five owned Go files — no trailing whitespace.
- Each command's `-h` exits 0 and `-count=0` is rejected before any Kafka operation.
- Producer unit tests cover defaults, rejection of a fixed event ID for batches, unique generated event/order IDs, and an explicitly shared order ID.

Live smoke against the already-running lab projects:

- Kafka 4.3.1 KRaft: producer wrote three records and sequential consumer read them; the sequential failpoint returned nonzero before commit and a retry read the same offset; transaction failpoint returned nonzero after output enqueue, retry committed two records, and a read-committed consumer read both outputs.
- Parallel consumer: after the shared runner cancellation-order fix, read 12 produced records across partitions 0, 1, and 2; produced and consumed event-ID sets matched.
- Kafka 3.9.2 ZooKeeper: producer wrote one record and consumer read it.
- During the first parallel smoke, the shared runner canceled its batch context before `CommitRecords`; the Lead assigned that runner fix to the Go worker. The smoke passed after the fix. No `internal/kafka` files were changed here.
- Existing project state after smoke: KRaft+PostgreSQL and ZooKeeper labs remain running; the three-node cluster project remains down. Smoke-created topics remain in the lab volumes.

## Blockers

None.


---

# T3b and registry rework report

Worktree: `/Users/andreigilmiiarov/.codex/worktrees/kafka-handbook/go-interview`

## Changed files

- `labs/kafka/internal/store/store.go`
- `labs/kafka/internal/store/store_integration_test.go`
- `labs/kafka/sql/schema.sql`
- `labs/kafka/sql/schema.go`
- `labs/kafka/cmd/outbox/main.go`
- `labs/kafka/cmd/outbox/main_test.go`
- `labs/kafka/cmd/outbox/main_integration_test.go`
- `site/scripts/content.mjs`
- `site/tests/kafka.test.mjs`

## Implementation

- PostgreSQL schema supports orders, pending outbox records, processed-event markers, and order-total projections. `CreateOrder` inserts the order and outbox record in one transaction. `ApplyEvent` writes the dedup marker and projection in one transaction. Pending publishing locks an ordered batch, uses a 10-second bounded send callback, marks only confirmed sends, rolls back at the approved premark failpoint, and rejects batch limits above 1000. Transaction rollback cleanup is bounded to five seconds.
- The outbox CLI implements `init`, `create`, `publish`, and `consume`; exposes a 60-second default `-timeout`; creates the topic before publishing; uses three partitions and min ISR 1 for RF=1 or 2 otherwise; consumes from the earliest offset with `read_committed`; commits offsets after PostgreSQL processing; and leaves the group with a bounded context. Main-process logs use JSON slog fields for topic/group/event/order IDs and the dedupe result.
- Integration tests use a unique PostgreSQL schema, topic, group, and IDs, then clean only those resources. The real binary smoke runs init/create, confirms exit 1 at publish and consume failpoints, succeeds on both retries, and verifies total=1285, processed count=1, pending count=0. The Kafka records show the same outbox event confirmed at offsets 0 and 1.
- Local Markdown references now resolve directories only through actual `index.md` or `README.md` files. Empty directories are reported unresolved. Malformed percent escapes produce a generic diagnostic without echoing the target.
- Added source snippet regions `create-order`, `publish-pending`, and `apply-event` in `store.go`.

## Validation

- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go test ./...` — PASS.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go build ./...` — PASS.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go vet ./...` — PASS.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go test -race -tags=integration -count=1 ./internal/store` — PASS; all five live PostgreSQL/Kafka integration cases passed, including 24 concurrent duplicate deliveries, projection rollback, and real outbox replay.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go test -race -tags=integration -count=1 -v ./cmd/outbox` — PASS; built executable smoke and three command tests passed.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go test -race ./cmd/outbox ./internal/store ./sql` — PASS.
- `env GOCACHE=/tmp/kafka-handbook-t3b-go-cache go build -o /tmp/kafka-handbook-t3b-outbox ./cmd/outbox` — PASS.
- `npm run test:site` — PASS; 37/37 tests.
- `npm run docs:check` — PASS; 8 groups, 42 Go topics, 20 exercises, and 8 Kafka topics.
- `git diff --check && git diff --cached --check` — PASS.
- `gofmt -w cmd/outbox/main.go cmd/outbox/main_test.go cmd/outbox/main_integration_test.go internal/store/store.go internal/store/store_integration_test.go sql/schema.go` — PASS.
- Test-first registry regression run `node --test site/tests/kafka.test.mjs` — RED as expected: both new cases reproduced the false negatives (13 passed, 2 failed). After the fix, the same command passed 15/15.

The initial sandboxed local-service integration run was denied network access; the identical approved local PostgreSQL/Kafka tests were rerun with escalation and passed. No dependencies were added here; no files were staged or committed.

## Assumptions and deviations

None.


---

# Kafka handbook implementation report

Implemented Russian Kafka overview and eight chapters, connected the Kafka registry to VitePress navigation/catalog, added the Kafka landing link and roadmap, and wrote the canonical runnable `labs/kafka/README.md`. The lab page includes that README as rendered Markdown. Source examples use named regions from the pinned `franz-go v1.22.1` implementation.

## Validation

- `npm run test:site` — PASS, 37/37 tests.
- `npm run docs:check` — PASS, 8 groups, 42 Go topics, 20 exercises, 8 Kafka topics.
- `npm run docs:build` — PASS; VitePress reported its existing bundle-size warning for chunks over 500 kB.
- `for c in producer consumer parallel-consumer transactions; do go run "./cmd/$c" -h; done` — PASS for all four CLIs.
- `for s in init create publish consume; do go run ./cmd/outbox "$s" -h; done` — PASS for all four outbox subcommands.
- `git diff --check && git diff --cached --check` — PASS.
- Generated `site/.vitepress/dist/kafka/lab.html` — exactly one H1 (`Лаборатория Kafka`), rendered README sections, no raw include directive. The three inspected chapter pages each have one H1 and omit the CLI parser dump.

No files were staged or committed.


## Итоговая приёмка Lead

Последнее исправление: все три пути ошибки подготовки PostgreSQL-схемы используют свежий context с тайм-аутом 5 s. `go test -race -tags integration -count=1 ./internal/store` — PASS после исправления; `gofmt -w internal/store/store_integration_test.go` — exit 0. Lead проверил код исправлений producer и cleanup; задачи ACCEPTED.

- `bash labs/kafka/scripts/lab.sh kraft down` — PASS, контейнеры и сеть удалены, volumes сохранены.
- `bash labs/kafka/scripts/lab.sh zk down` — PASS, контейнеры и сеть удалены, volumes сохранены.
- `docker ps --format '{{.Names}}'` — exit 0, пустой вывод.
- `docker volume ls --format '{{.Name}}'` — exit 0, все восемь named volumes трёх учебных проектов присутствуют.
- `git diff --check` и `git diff --cached --check` — PASS.
- Проверка каждого нового файла через `git diff --no-index --check /dev/null <file>` — без замечаний.
- Worktree staging пуст; исходный checkout сохраняет ранее staged `docs/active-plan.md`. Изменения реализации находятся в worktree kafka-handbook.
