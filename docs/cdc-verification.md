# Проверка CDC-лаборатории

Дата отчёта: 2026-09-30. **Статус: COMPLETE; принято Lead.** Полные E2E-проверки обеих сред прошли после увеличения максимального heap Kafka Connect до `1GiB`; выбранные image и digest не менялись.

## Среды и версии

- Основная среда: Kafka broker `4.3.1` в KRaft, PostgreSQL `17.6`, Kafka Connect runtime `4.3.0`, PostgreSQL plugin Debezium `3.6.3.Final`.
- Совместимая среда: Kafka broker `3.9.2` и ZooKeeper `3.9.3`; Connect runtime и PostgreSQL plugin сообщают те же версии.
- Обе среды используют `quay.io/debezium/connect:3.6.3@sha256:5638e8e42c6681d1dfd753324040b7949e3cfe3f8f3aff1e39fc8c0b63c6ccec` и отдельные Compose projects/volumes. Connect worker задан `-Xms512m -Xmx1g`, чтобы JVM могла сканировать bundled plugins.

## Compose и Connect

Для инфраструктуры проверены команды:

```bash
bash labs/kafka/cdc/lab.sh kraft up
bash labs/kafka/cdc/lab.sh kraft status
bash labs/kafka/cdc/lab.sh zk up
bash labs/kafka/cdc/lab.sh zk status
docker compose -p kafka-cdc-kraft -f labs/kafka/cdc/compose.kraft.yaml config --quiet
docker compose -p kafka-cdc-zk -f labs/kafka/cdc/compose.zk.yaml config --quiet
```

`up`/`status` прошли для обеих сред. Connect connector/task находились в `RUNNING`; source offsets и активные replication slots были доступны. Ready LSN при старте: `29940488` для `kraft`, `26424232` для `zk`; worker/plugin versions указаны выше. Проверка JSON connector configuration и outbox topic predicate также прошла. Подробный инфраструктурный журнал сохранён в `/tmp/kafka-handbook-execution/cdc-infra-report.md`.

Для Compose-файлов выполнен `docker compose ... config --quiet` выше. Скрипт `bash -n labs/kafka/cdc/lab.sh`, JSON parse и проверки routing/predicate прошли. Docker/localhost проверки выполнялись с разрешённым локальным доступом: обычный sandbox блокировал Docker socket и localhost integration sockets.

## Модель, consumer и Go

Из `labs/kafka`:

```bash
GOCACHE=/tmp/cdcmodel-go-build-cache go test ./internal/cdcmodel
CDC_DATABASE_URL='postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable' \
CDC_ADMIN_DATABASE_URL='postgres://cdc_admin:cdc_admin@127.0.0.1:25432/cdc_lab?sslmode=disable' \
GOCACHE=/tmp/cdcmodel-go-build-cache \
  go test -race -tags integration ./internal/cdcmodel
GOCACHE=/tmp/kafka-cdc-gocache go test -race -tags=integration ./internal/cdcconsumer -count=1 -v
CDC_KAFKA_BROKERS=127.0.0.1:59092 \
CDC_DATABASE_URL='postgres://cdc_app:cdc_app@127.0.0.1:35432/cdc_lab?sslmode=disable' \
CDC_ADMIN_DATABASE_URL='postgres://cdc_admin:cdc_admin@127.0.0.1:35432/cdc_lab?sslmode=disable' \
GOCACHE=/tmp/kafka-cdc-gocache \
  go test -race -tags=integration ./internal/cdcconsumer -count=1 -v
GOCACHE=/tmp/kafka-cdc-gocache go test ./...
GOCACHE=/tmp/kafka-cdc-gocache go test -race ./...
GOCACHE=/tmp/kafka-cdc-gocache go build ./...
```

Model unit tests и isolated-schema model integration race test прошли на основной PostgreSQL среде; этот тест требует admin URL и повторяет ограниченные права production outbox. Consumer integration race tests прошли с обоими Kafka brokers. Они проверяют malformed payload, неправильный Kafka key, пропуск и старую версию заказа, конфликт event ID, безопасную повторную доставку после commit БД и независимый replay.

Полные `go test ./...`, `go test -race ./...` и `go build ./...` из `labs/kafka` прошли. Корневые `GOCACHE=/tmp/kafka-cdc-gocache go test ./...` и `GOCACHE=/tmp/kafka-cdc-gocache go build ./...` также завершились с exit code `0`. Первая попытка корневого `httptest` в sandbox завершилась `operation not permitted` на localhost bind; повтор с разрешённым доступом прошёл, это не ошибка Go-кода. CDC не добавил Go-зависимостей.

## Сквозные сценарии

Smoke `1000 → 1500 → 1200` достиг projection version `3`; повтор `create` с тем же event ID после последующих обновлений вернул исходное событие. Полный verifier запускает три сценария: заказ и replay команды, публикацию записей при перезапуске Connect и восстановление после перезапуска broker.

| Среда | Команда | Результат |
| --- | --- | --- |
| Kafka `4.3.1`/KRaft | `GOCACHE=/tmp/cdcmodel-go-build-cache bash cdc/verify.sh kraft > /tmp/cdc-lab-execution/e2e-kraft-heap1g.log 2>&1` | PASS, `internal/cdce2e` за `16.259s`; ready LSN `31524520` |
| Kafka `3.9.2`/ZooKeeper | `GOCACHE=/tmp/cdcmodel-go-build-cache bash cdc/verify.sh zk > /tmp/cdc-lab-execution/e2e-zk-heap1g.log 2>&1` | PASS, `internal/cdce2e` за `27.289s`; ready LSN `29676840` |

Во время предыдущего `verify.sh zk` Connect restart не прошёл: Compose вернул управление до готовности Java worker, а сканирование bundled plugins завершилось `java.lang.OutOfMemoryError: Java heap space` при `-Xmx512m`; readiness не получила Connect REST. Это JVM heap failure, не broker TLS и не kernel OOM (`OOMKilled=false`). EXIT cleanup сценария восстановил выбранные broker и Connect services. Увеличение максимального heap Connect до `1GiB` в обоих Compose-файлах устранило причину; образ и digest сохранены. После корректировки обе полные E2E-проверки выше прошли. Исторические diagnostics: `/tmp/cdc-lab-execution/e2e-zk.log` и `/tmp/cdc-lab-execution/compat-connect.log`.

Промежуточные логи: `/tmp/cdc-lab-execution/e2e-kraft.log`, `/tmp/cdc-lab-execution/e2e-kraft-broker-rerun.log`, `/tmp/cdc-lab-execution/e2e-zk.log`, `/tmp/cdc-lab-execution/compat-connect.log`. Сводка модели и E2E: `/tmp/cdc-lab-execution/e2e-report.md`.

## Сайт и Git

- `npm run test:site` — PASS, 37/37.
- `npm run docs:build` — PASS; VitePress отрендерил страницы. Сборка предупредила о чанках размером более `500 kB` после minification.
- `npm run docs:check` — FAIL, 35 существующих broken relative links из миграционных документов/примеров. Такой же набор из 35 ошибок был зафиксирован до CDC; CDC-ссылки и новые страницы ошибок не добавили. Каталоги находятся вне scope текущей работы.
- Браузерная проверка preview `4174` — PASS: поиск находит CDC-лабораторию; sidebar содержит ссылку на неё и все 8 Kafka глав; на странице один `h1`; три details закрыты по умолчанию, второй ответ открывается Space; при viewport `390×844` горизонтальной прокрутки нет (`scrollWidth=390`).
- `git diff --check`, `git diff --cached --check` и проверка новых файлов через `git diff --no-index --check` — PASS.

После проверки `bash labs/kafka/cdc/lab.sh kraft down` и `bash labs/kafka/cdc/lab.sh zk down` завершились успешно. Проверка Docker показала пустые списки контейнеров для обоих Compose projects, при этом сохранены все три `kafka-cdc-kraft_*` volumes и все пять `kafka-cdc-zk_*` volumes. Ни один CDC volume не удалялся.

Полный лог `npm run docs:check` (35 pre-existing ошибок):

<details>
<summary>Показать полный список unresolved paths</summary>

```text
README.md: unresolved relative link path "docs/migration.md".
docs/content-authoring.md: unresolved relative link path "migration-language.md".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/line_validation.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/logging.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/client_interfaces.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/user_handlers.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/user_handlers.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/user_handlers.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/user_api_client.go".
docs/migrations/migration-backend.md: unresolved relative link path "../examples/backend/backend_test.go".
docs/migrations/migration-language.md: unresolved relative link path "../practice/nil-empty-json/README.md".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/slice_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/range_mutation.go".
docs/migrations/migration-language.md: unresolved relative link path "../practice/range-mutation/README.md".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/slice_layout.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/slice_layout_bench_test.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/slice_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/slice_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/defer_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/defer_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/defer_examples.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/user_error.go".
docs/migrations/migration-language.md: unresolved relative link path "../practice/typed-nil/README.md".
docs/migrations/migration-language.md: unresolved relative link path "../practice/typed-nil/README.md".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/user_error.go".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/select_drain.go".
docs/migrations/migration-language.md: unresolved relative link path "../practice/select-states/README.md".
docs/migrations/migration-language.md: unresolved relative link path "../examples/language/select_drain.go".
docs/migrations/migration-language.md: unresolved relative link path "../practice/select-states/README.md".
```

</details>
