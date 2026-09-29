# CDC-лаборатория: Debezium и Kafka Connect

Эта лаборатория показывает отдельный путь из PostgreSQL outbox в Kafka через logical decoding, Debezium и Kafka Connect. Существующая лаборатория transactional outbox продолжает использовать polling; CDC не переключает и не мигрирует её. Это изолированная учебная среда с демонстрационными credentials, без Schema Registry, HA-топологии и production-настроек.

## Требования

Нужны Go версии, указанной в `labs/kafka/go.mod`, Docker Engine с Docker Compose plugin v2, Bash, `curl` и `python3`. Скрипт `lab.sh` использует их для запуска Compose, REST-проверок и проверки настроек topics. Дополнительные инструменты и Go-зависимости для CDC устанавливать не требуется.

## Среды и запуск

| Режим | Kafka broker | Kafka Connect | Хостовые адреса | Compose project и данные |
| --- | --- | --- | --- | --- |
| `kraft` | `4.3.1`, KRaft | Образ Debezium `3.6.3`; REST сообщает Kafka Connect `4.3.0`, plugin `3.6.3.Final` | broker `127.0.0.1:49092`, PostgreSQL `127.0.0.1:25432`, Connect REST `127.0.0.1:18083` | Отдельный изолированный project и volumes |
| `zk` | `3.9.2`, ZooKeeper | Тот же образ и версии Connect runtime/plugin; consumer integration и полный E2E пройдены | broker `127.0.0.1:59092`, PostgreSQL `127.0.0.1:35432`, Connect REST `127.0.0.1:28083` | Другой project и volumes, не общие с `kraft` или polling |

Версия Connect runtime не совпадает с версией broker: в основной среде это `4.3.0` против Kafka `4.3.1`. Connect REST обеих сред сообщает plugin `io.debezium.connector.postgresql.PostgresConnector` версии `3.6.3.Final`. Compose закрепляет образ `quay.io/debezium/connect:3.6.3` digest `sha256:5638e8e42c6681d1dfd753324040b7949e3cfe3f8f3aff1e39fc8c0b63c6ccec`; для Connect задан heap `-Xms512m -Xmx1g`, чтобы JVM могла просканировать bundled plugins. Инфраструктурная readiness, consumer integration и полные E2E/fault-сценарии пройдены для обеих сред. Подробные результаты, включая initial OOM и исправление, записаны в `docs/cdc-verification.md` репозитория.

Запускайте команды из `labs/kafka`:

```bash
bash cdc/lab.sh kraft up
bash cdc/lab.sh kraft ready
bash cdc/lab.sh kraft status
```

Поддерживаются действия `up`, `ready`, `status`, `logs`, `down` и `reset` для обоих режимов. `down` останавливает и удаляет контейнеры, сохраняя volumes; `reset` явно удаляет данные выбранного CDC project. Если сохранённое состояние несовместимо с конфигурацией, запуск сообщает об ошибке и не переписывает его молча. Сначала запустите только выбранный режим.

Команда `up` создаёт topic `cdc.orders`, compact heartbeat topic `__debezium-heartbeat.cdc` и connector, затем ждёт readiness. Рабочий Connect хранит свои конфигурацию, offsets и status в отдельных внутренних topics: `cdc-kraft-connect-configs`, `cdc-kraft-connect-offsets`, `cdc-kraft-connect-status` для `kraft` и соответствующих `cdc-zk-connect-*` для `zk`. Heartbeat topic нужен для подтверждения продвижения source LSN, в том числе когда outbox пока не получает новых строк. Это topics состояния и служебного прогресса Connect/Debezium; Schema Registry в лаборатории нет.

`ready` ждёт состояния `RUNNING` у connector и всех его tasks, а также сохранённого ненулевого source LSN после завершения начального snapshot. Эта проверка подтверждает готовность connector, но не заменяет end-to-end сценарий заказа и projection. После чистого первичного запуска его можно проверить отдельным интеграционным сценарием:

```bash
bash cdc/verify.sh kraft
```

Скрипт работает с выбранной Compose-средой, выполняет Go integration tests с перезапуском Connect и broker и не делает `reset`; оставленные данные и volumes сохраняются.

В обеих средах используется отдельная база `cdc_lab`. Учебные параметры подключения с хоста:

| Роль | Имя | Пароль | Назначение |
| --- | --- | --- | --- |
| Приложение | `cdc_app` | `cdc_app` | Команды заказа и projection |
| Администрирование | `cdc_admin` | `cdc_admin` | Создание схемы и лабораторные операции |
| Репликация | `cdc_replication` | `cdc_replication` | Logical replication для Debezium |

На host-порте `25432` строка приложения выглядит так:

```text
postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable
```

Для `zk` используйте порт PostgreSQL `35432`. Go-команды читают `CDC_DATABASE_URL`, `CDC_KAFKA_BROKERS` и `CDC_TOPIC`. Значения основной среды:

```bash
export CDC_DATABASE_URL='postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable'
export CDC_KAFKA_BROKERS='127.0.0.1:49092'
export CDC_TOPIC='cdc.orders'
```

В режиме `zk` измените два адреса на `127.0.0.1:35432` и `127.0.0.1:59092`.

## Порядок событий и потока

`create/update` → одна PostgreSQL-транзакция (заказ + outbox INSERT) → WAL через `pgoutput` и replication slot → Debezium 3.6.3 + Outbox Event Router → `cdc.orders` (key `order_id`) → последовательный consumer → одна PostgreSQL-транзакция (inbox + projection) → commit Kafka offset.

Заказ и вставка outbox-события коммитятся атомарно. Таблица outbox в CDC-схеме insert-only; прикладная роль получает на неё `INSERT` и `SELECT`, но не `UPDATE` и `DELETE`. Logical decoding использует `pgoutput`, отдельные publication и slot, а replication-подключение выполняет роль `cdc_replication`. PostgreSQL connector передаёт change event с envelope `before`/`after`/`source`/`op`; Kafka Connect converter влияет на сериализованную форму. Outbox Event Router берёт настроенный payload и по умолчанию отправляет только его, без дополнительных envelope-полей. Официальные описания относятся к Debezium 3.6: [PostgreSQL connector](https://debezium.io/documentation/reference/3.6/connectors/postgresql.html) и [Outbox Event Router](https://debezium.io/documentation/reference/3.6/transformations/outbox-event-router.html).

Сокращённая иллюстрация connector change event **до** Router (не runtime record лаборатории): поля показывают outbox row `id`, `aggregatetype`, `aggregateid`, `type` и `payload`; JSONB `payload` здесь условно сериализован как escaped JSON string. Точная wire-форма зависит от Kafka Connect converter.

```json
{
  "before": null,
  "after": {
    "id": "evt-example-create",
    "aggregatetype": "orders",
    "aggregateid": "ord-example",
    "type": "order.created",
    "payload": "{\"event_id\":\"evt-example-create\",\"order_id\":\"ord-example\",\"type\":\"order.created\",\"version\":1,\"order_version\":1,\"amount_cents\":1000}"
  },
  "source": {
    "connector": "postgresql",
    "db": "cdc_lab",
    "schema": "public",
    "table": "cdc_outbox",
    "snapshot": false
  },
  "op": "c",
  "ts_ms": 0
}
```

Пример JSON payload после Router:

```json
{
  "event_id": "evt-example-create",
  "order_id": "ord-example",
  "type": "order.created",
  "version": 1,
  "order_version": 1,
  "amount_cents": 1000
}
```

`version` — версия формата события и всегда равна `1`; `order_version` — версия заказа. Создание имеет тип `order.created` и версию заказа `1`; каждое успешное обновление имеет тип `order.updated` и следующую версию. `amount_cents` хранит абсолютную сумму, а не delta. Kafka key равен `order_id`. Тема `cdc.orders` имеет три partitions и `cleanup.policy=delete` с retention семь дней.

## Сквозной сценарий: 1000 → 1500 → 1200

Для чистого первичного сценария создайте пустую среду явным `reset`, затем выполните `up`; он создаст новую БД с пустым outbox. `up` и `down` сами не очищают сохранённые volumes. Если начальный snapshot видит накопленные строки outbox, connector опубликует их как snapshot-записи, но порядок таких строк не гарантирует историческую последовательность изменений заказа. Поэтому snapshot накопленной истории не заменяет replay последовательности событий: consumer может обнаружить неизвестную старую версию или gap. Не выполняйте `reset` автоматически при повторном использовании volumes: сначала проверьте состояние и решите, нужно ли удалять эти данные. Topic `cdc.orders` хранит до семи дней истории, в том числе записи предыдущих запусков и readiness probe; у новой consumer group чтение начинается с доступной истории.

Флаг `-count` считает Kafka records, включая дубликаты, старые записи других заказов и probe, а не изменения одного заказа. Используйте одно новое имя projection на весь walkthrough и после каждого `consume` проверяйте версию именно целевого заказа. Если запись ещё не дошла до projection, обработайте следующую Kafka record тем же projection и проверьте состояние снова. Не пересоздавайте projection между шагами, иначе новая group снова начнёт читать retained history.

Сначала создайте заказ. Команда `create` возвращает полное JSON-событие:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
order_id="ord-$run_id"
projection="walkthrough-$run_id"
go run ./cmd/cdc create \
  -event-id "create-$run_id" \
  -order-id "$order_id" \
  -amount-cents 1000
```

Обработайте одно событие и прочитайте источник и projection. `order` возвращает текущее состояние заказа; `projection` возвращает независимо построенное состояние consumer. Оба ответа имеют JSON-форму `{"order_id":"…","order_version":1,"amount_cents":1000}`:

```bash
go run ./cmd/cdc consume -projection "$projection" -count 1
go run ./cmd/cdc order -order-id "$order_id"
go run ./cmd/cdc projection -projection "$projection" -order-id "$order_id"
```

Теперь обновите абсолютную сумму с ожидаемой версией `1`:

```bash
go run ./cmd/cdc update \
  -event-id "update-2-$run_id" \
  -order-id "$order_id" \
  -expected-version 1 \
  -amount-cents 1500
go run ./cmd/cdc consume -projection "$projection" -count 1
go run ./cmd/cdc projection -projection "$projection" -order-id "$order_id"
```

Если состояние ещё имеет версию `1`, повторяйте две последние команды в том же projection, пока не увидите `order_version=2`. Затем создайте второе обновление с ожидаемой версией `2`:

```bash
go run ./cmd/cdc update \
  -event-id "update-3-$run_id" \
  -order-id "$order_id" \
  -expected-version 2 \
  -amount-cents 1200
go run ./cmd/cdc consume -projection "$projection" -count 1
go run ./cmd/cdc projection -projection "$projection" -order-id "$order_id"
```

Если после второго обновления состояние ещё имеет версию `2`, снова повторяйте `consume` и `projection` с тем же projection, пока не увидите `order_version=3`. Ожидаемое конечное состояние — `amount_cents=1200`. `create` и `update` возвращают полное событие; `order` и `projection` возвращают только объект состояния. Projection не читает исходный outbox и не присоединяет его к заказу, поэтому команда чтения не синтезирует `event_id`.

Обновление использует optimistic version check: при двух командах с одним `expected-version` выигрывает не больше одной. `event_id` также является ключом идемпотентности команды. Повтор того же ID с теми же нормализованными параметрами возвращает исходное событие, даже если заказ уже продвинулся дальше; тот же ID с изменёнными параметрами отклоняется как конфликт.

## Дубликаты и replay

Consumer читает последовательно с ручным commit. Одна DB-транзакция проверяет inbox ID и payload hash, проверяет `order_version`, затем обновляет inbox и projection. Kafka offset фиксируется только после commit БД. Повтор известного `event_id` с тем же payload пропускается. Payload с другим hash под тем же ID, неизвестное старое событие, пропуск версии, повреждённый JSON или key, не равный `order_id`, останавливают обработку; offset за проблемной записью не фиксируется.

Чтобы показать окно между DB commit и Kafka offset commit, выполните consumer с failpoint, затем повторите ту же projection:

```bash
go run ./cmd/cdc consume -projection retry -count 1 -fail-after-process-before-commit
go run ./cmd/cdc consume -projection retry -count 1
```

Первый вызов намеренно возвращает ошибку. Повторная доставка безопасна: inbox и projection уже атомарно сохранили эффект. Параметр `-timeout` задаёт общий срок ожидания; значение по умолчанию — `60s`. `-count` по умолчанию равен `1`.

Для независимого replay выберите новое имя projection. Для каждой projection используется своя consumer group `cdc.<projection>` и отдельная inbox; это позволяет пересобрать состояние независимо от текущего consumer. Replay ограничен retention темы в семь дней: если начало последовательности заказа уже удалено, неизвестная старая версия или gap не должны тихо менять projection.

## Snapshot, диагностика и восстановление

При первом подключении connector снимает snapshot настроенных таблиц; для чистого walkthrough outbox должен быть пуст на этот момент. `ready` подтверждает `RUNNING` connector и tasks и записанный source LSN после snapshot, но не проверяет доставку конкретного заказа. Для end-to-end проверки используйте команду `verify.sh` выше. Команда `status` показывает состояние connector/task, offsets connector, lag consumer groups, активность replication slot и удерживаемый WAL:

```bash
bash cdc/lab.sh kraft status
bash cdc/lab.sh kraft logs
```

Если connector остановить или Kafka станет недоступна, slot может удерживать WAL. Для среды задан `max_slot_wal_keep_size=256MiB`; PostgreSQL проверяет этот порог на checkpoint. Это ограничение slot-ов, а не жёсткий потолок дискового WAL: при превышении нужные сегменты могут быть удалены и slot потеряет возможность продолжить со старого LSN. Следите за диском и сверяйте состояние slot по [документации PostgreSQL 17](https://www.postgresql.org/docs/17/runtime-config-replication.html). Потеря slot или offsets, ручная очистка и запуск нового snapshot — отдельные recovery-действия, а не обычное продолжение. Повторный snapshot не восстанавливает потерянную историческую последовательность. Для полного replay создайте отдельные projection/group/inbox, пока необходимые события ещё находятся в теме.

После недоступности broker consumer может завершиться с сетевой ошибкой; бесконечные автоматические повторы здесь не обещаются. Когда Kafka снова доступна, проверьте `status`. Если Connect task имеет состояние `FAILED`, перезапустите только failed tasks штатным REST endpoint, не удаляя connector, offsets или slot; затем дождитесь `ready`. Если consumer завершился, запустите его заново с тем же `-projection`: он продолжит из offset той же consumer group. Например, для `kraft`:

```bash
bash cdc/lab.sh kraft status
curl --fail-with-body -X POST \
  'http://127.0.0.1:18083/connectors/cdc-orders/restart?includeTasks=true&onlyFailed=true'
bash cdc/lab.sh kraft ready
go run ./cmd/cdc consume -projection "$projection" -count 1
```

Для `zk` замените REST endpoint на `http://127.0.0.1:28083` и перед командой восстановления убедитесь, что выбранная Compose-среда запущена. После восстановления снова проверяйте projection и consumer-group lag; не удаляйте состояние для обхода ошибки.

Этот сценарий объясняет CDC как альтернативу polling. Он не переключает polling publisher, не задаёт cutover/replay-план для production и не обещает сохранность всей истории после истечения retention. Smoke-сценарий заказа и повторной команды, инфраструктурные проверки и полные E2E/fault-сценарии обеих сред прошли; подробности находятся в `docs/cdc-verification.md` репозитория.

## Самопроверка

<details>
<summary>Почему перед созданием первого заказа важно дождаться завершения пустого initial snapshot?</summary>

Так начальный snapshot не будет выдавать накопленные строки как будто это текущая последовательность изменений. После подтверждения пустого snapshot новый end-to-end probe проверяет путь от свежей записи outbox через WAL и Router до consumer.

</details>

<details>
<summary>Почему известный повтор можно пропустить, а старую неизвестную версию нужно остановить?</summary>

Inbox подтверждает, что тот же event ID и payload уже применялись. Неизвестная старая версия может означать, что история уже потеряна или пришла вне порядка; без подтверждённого inbox нельзя считать её безопасным дубликатом.

</details>

## Практика

Остановите Connect до обновления заказа, выполните несколько обновлений, затем восстановите connector. Проверьте состояние task, lag и удерживаемый WAL; убедитесь, что `order_version` в projection идёт последовательно и offset не проходит проблемное событие.

<details>
<summary>Решение: connector восстановился, но projection сообщила gap. Что проверить?</summary>

Проверьте сохранённые Kafka offsets Connect и состояние replication slot, выясните, не был ли slot потерян или пересоздан, и сопоставьте WAL с `max_slot_wal_keep_size=256MiB`, учитывая, что PostgreSQL проверяет этот порог на checkpoint. Это не лимит всего дискового WAL. Не удаляйте slot/offsets для «починки» работающего состояния. Для чистого replay используйте пустые projection, inbox и новую group, и только если требуемая последовательность всё ещё удерживается в теме.

</details>

## Проверки

Для Go-модуля запускайте из `labs/kafka`:

```bash
go test ./...
go test -race ./...
go build ./...
```

Интеграционные сценарии используют Compose среды `kraft` или `zk`. Команда `down` останавливает выбранную среду и сохраняет лабораторное состояние; `reset` удаляет volumes только по явному решению.

Полный набор E2E/fault-проверок запускается из `labs/kafka` командой `bash cdc/verify.sh kraft` или `bash cdc/verify.sh zk`. Для остановки выбранной среды с сохранением volumes выполните `bash cdc/lab.sh <kraft|zk> down`.
