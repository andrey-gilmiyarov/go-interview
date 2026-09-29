# Лаборатория Kafka

Отдельная лаборатория для глав в `site/kafka/`. Docker Compose запускает брокеры и, при необходимости, PostgreSQL; Go-клиенты и тесты запускаются на хосте из этого модуля. Учебный контракт события намеренно узкий: `order.created`, версия `1`, положительный `amount_cents`; decoder отклоняет неизвестные поля и данные после JSON-объекта.

Нужны Docker Compose и Go toolchain `1.27.1` (версия закреплена в `go.mod`). Три брокера используют по 512 MiB heap каждый, поэтому кластерный режим требует больше 1.5 GiB свободной памяти с учётом самого Docker и ОС. Все опубликованные порты привязаны к `127.0.0.1`; listeners используют PLAINTEXT внутри учебной сети Docker. Это локальная среда с демонстрационными credentials, не production-конфигурация.

## Режимы и порты

| Режим | Брокер / координация | Адрес с хоста | Для чего |
| --- | --- | --- | --- |
| `kraft` | Kafka `4.3.1`, один совмещённый broker/controller, KRaft | `localhost:19092` | Основные примеры; PostgreSQL для outbox добавляется через `--outbox` |
| `zk` | Kafka `3.9.2` и ZooKeeper `3.9.3` | `localhost:29092` | Сверка базовых consumer/producer сценариев на ZooKeeper mode |
| `cluster` | Kafka `4.3.1`, три совмещённых broker/controller, KRaft | `localhost:39092`, `39093`, `39094` | RF=3, `min.insync.replicas=2` и отказ одного broker |

В режиме `kraft` опциональный PostgreSQL `17.6` доступен на `localhost:15432`: база, пользователь и учебный пароль — `kafka_lab`. Он не запускается без профиля outbox.

Из корня репозитория запустите базовую среду:

```bash
bash labs/kafka/scripts/lab.sh kraft up
```

Чтобы добавить PostgreSQL:

```bash
bash labs/kafka/scripts/lab.sh kraft up --outbox
```

Скрипт ожидает healthcheck-ы до 180 секунд. Положительное целое число секунд можно задать через `KAFKA_LAB_WAIT_TIMEOUT`. У каждой среды свой Compose project и именованные volumes; состояния трёх режимов не смешиваются.

## Быстрый producer и consumer

Из корня репозитория запустите Go-модуль; следующие примеры Go предполагают, что текущий каталог остаётся `labs/kafka`:

```bash
cd labs/kafka
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="orders-quick-$run_id"
go run ./cmd/producer -count 1 -topic "$topic" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 12500
go run ./cmd/consumer -count 1 -topic "$topic" -group "quick-$run_id"
```

Программы по умолчанию используют брокер `localhost:19092`, topic `orders`, group `orders-lab` и replication factor `1`. В примере выше отдельные topic и group не смешиваются с данными предыдущих запусков. Новая группа начинает чтение с начала своего топика, существующая продолжает с committed position. Consumer отключает auto-commit, начинает с `AtStart` при отсутствии committed position и использует `read_committed`. Приложения создают topic на три партиции, если его ещё нет.

Чтобы увидеть повтор после эффекта и до commit Kafka, используйте отдельную тему и группу. Failpoint-команда ожидаемо завершается с ошибкой; запустите команды по одной, затем выполните повтор тем же group ID:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="orders-retry-$run_id"
group="retry-$run_id"
go run ./cmd/producer -count 1 -topic "$topic" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 700
go run ./cmd/consumer -count 1 -topic "$topic" -group "$group" -fail-after-process-before-commit
go run ./cmd/consumer -count 1 -topic "$topic" -group "$group"
```

`event_id` описывает бизнес-событие, `order_id` становится Kafka key. Не задавайте один `-event-id` для `-count` больше одного: producer отклоняет такое сочетание, чтобы одна идентичность не изображала несколько событий. Первый запуск consumer намеренно завершается через failpoint после обработки и до commit; второй запуск той же группы получает ту же запись. Этот CLI-сценарий показывает повторную доставку и сам по себе не записывает внешний бизнес-эффект в PostgreSQL.

## Остальные сценарии

Параллельный consumer ограничивает конкурентность партициями, сохраняя последовательную обработку внутри каждой партиции. В отдельной теме отправьте три записи, затем прочитайте их той же группой:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="orders-parallel-$run_id"
go run ./cmd/producer -count 3 -topic "$topic"
go run ./cmd/parallel-consumer -count 3 -topic "$topic" -group "parallel-$run_id"
```

Kafka-to-Kafka transaction читает входную тему, пишет выход и связывает output с offset входной группы одной транзакцией. Для каждого прогона возьмите отдельные темы, group и transactional ID:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
input="orders-transform-$run_id"
output="orders-processed-$run_id"
go run ./cmd/producer -count 1 -topic "$input" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 12500
go run ./cmd/transactions -input-topic "$input" -output-topic "$output" -group "transform-$run_id" -transactional-id "transform-$run_id" -count 1
go run ./cmd/consumer -count 1 -topic "$output" -group "output-$run_id"
```

Для проверки abort создайте отдельный input и запустите transform с `-fail-after-output`; эта команда намеренно завершится с ошибкой. Затем проверьте выход новой группой с коротким timeout:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
input="orders-abort-$run_id"
output="orders-aborted-$run_id"
go run ./cmd/producer -count 1 -topic "$input" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 12500
go run ./cmd/transactions -input-topic "$input" -output-topic "$output" -group "abort-$run_id" -transactional-id "abort-$run_id" -count 1 -fail-after-output
go run ./cmd/consumer -count 1 -timeout 2s -topic "$output" -group "abort-check-$run_id"
```

Последняя команда с `read_committed` завершится по timeout, потому что aborted output не выдаётся. Не считайте это доказательством, что внешний DB/HTTP эффект тоже откатился: такого ресурса в Kafka transaction нет.

### PostgreSQL transactional outbox

Из каталога `labs/kafka` запустите профиль PostgreSQL:

```bash
bash scripts/lab.sh kraft up --outbox
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="orders-outbox-$run_id"
event_id="evt-$run_id"
order_id="ord-$run_id"
go run ./cmd/outbox init
```

`DATABASE_URL` необязателен; по умолчанию используется `postgres://kafka_lab:kafka_lab@localhost:15432/kafka_lab?sslmode=disable`. `create` сохраняет заказ и outbox row в одной PostgreSQL transaction. `publish` отправляет ограниченный batch один раз, ждёт подтверждения Kafka, затем фиксирует markers в БД. Чтобы failpoint действовал на ожидаемое событие, сначала убедитесь, что таблица не содержит чужих pending rows:

```bash
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec -T outbox-postgres psql -U kafka_lab -d kafka_lab \
  -c 'SELECT event_id FROM outbox WHERE published_at IS NULL ORDER BY created_at, event_id;'
```

Если запрос вернул старые строки, опубликуйте их обычной командой до сценария или сбросьте данные только после того, как они больше не нужны. Следующие команды используют уникальные topic, event ID и order ID:

```bash
go run ./cmd/outbox create -event-id "$event_id" -order-id "$order_id" -amount-cents 12500
KAFKA_TOPIC="$topic" go run ./cmd/outbox publish -limit 1 -fail-after-send-before-mark
KAFKA_TOPIC="$topic" go run ./cmd/outbox publish -limit 1
```

Первая команда `publish` намеренно возвращает ошибку после подтверждения Kafka, но до marker; повторный вызов публикует то же событие второй раз. `consume -count 1` применит DB-эффект и откажется до commit Kafka, следующий вызов `-count 2` прочитает повторную доставку и вторую Kafka-копию. Ни один из двух повторов не должен второй раз изменить агрегат:

```bash
KAFKA_TOPIC="$topic" go run ./cmd/outbox consume -consumer "order-totals-$run_id" -count 1 -fail-after-process-before-commit
KAFKA_TOPIC="$topic" go run ./cmd/outbox consume -consumer "order-totals-$run_id" -count 2
```

Проверьте два Kafka records и PostgreSQL состояние выбранного события:

```bash
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec broker /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server broker:19093 \
  --topic "$topic" --from-beginning --max-messages 2 --timeout-ms 10000
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec -T outbox-postgres psql -U kafka_lab -d kafka_lab \
  -c "SELECT event_id, published_at FROM outbox WHERE event_id = '$event_id';"
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec -T outbox-postgres psql -U kafka_lab -d kafka_lab \
  -c "SELECT consumer_id, event_id FROM processed_events WHERE event_id = '$event_id';"
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec -T outbox-postgres psql -U kafka_lab -d kafka_lab \
  -c "SELECT order_id, total_cents FROM order_totals WHERE order_id = '$order_id';"
```

`-consumer` задаёт Kafka group и стабильный namespace DB-дедупликации `(consumer_id, event_id)`; повторите его без изменений после failpoint. Каждая outbox-команда поддерживает `-timeout` со значением по умолчанию `60s`. `retry-topic`, DLQ и replay в CLI не реализованы.

## Consumer groups и offsets

Чтобы увидеть membership и assignment до поступления records, сначала сгенерируйте уникальный суффикс и скопируйте его в три терминала. Для каждого нового прогона используйте новый суффикс:

```bash
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
printf 'Скопируйте в три терминала: %s\n' "$run_id"
```

В каждом терминале присвойте `run_id` скопированному значению, затем используйте одну и ту же пару topic/group:

```bash
run_id="ВСТАВЬТЕ_СКОПИРОВАННЫЙ_СУФФИКС"
topic="orders-groups-$run_id"
group="groups-$run_id"
go run ./cmd/consumer -count 1 -timeout 5m -topic "$topic" -group "$group"
```

Сначала запустите эту команду в двух терминалах. Дождитесь двух участников в состоянии `Stable`, проверив команду диагностики ниже. Затем в третьем терминале присвойте тот же `run_id` и отправьте запись:

```bash
run_id="ВСТАВЬТЕ_СКОПИРОВАННЫЙ_СУФФИКС"
go run ./cmd/producer -count 1 -topic "orders-groups-$run_id" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 500
```

Kafka назначает три партиции между двумя активными членами группы; конкретная партиция записи и consumer, который её получит, зависят от key и assignment. Остановите оставшегося consumer после наблюдения. Если участников больше, чем партиций, лишние останутся без назначения. Разные приложения, которым нужно прочитать один поток независимо, должны использовать разные group IDs.

Для каждого примера `-count` ограничивает число обработанных записей; `-timeout` задаёт общий срок ожидания и по умолчанию равен `60s`. У producer при истечении application wait результат может быть неизвестен: timeout не доказывает отсутствие записи. Сохраняйте event ID при повторе и проверяйте потребителя на дубликаты.

## Репликация и failover

Кластерный сценарий проверяет один broker отказа. Скрипт создаёт отдельную временную тему RF=3/minISR=2, подтверждает запись до остановки `broker-1`, останавливает один узел, проверяет KRaft leader и чтение/подтверждение новой записи, затем пытается восстановить broker и удалить временную тему:

Для отдельной темы с RF=3 запустите кластер и задайте replication factor в конфигурации Go-клиента:

```bash
bash scripts/lab.sh cluster up
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
topic="orders-rf3-$run_id"
KAFKA_BROKERS=localhost:39092,localhost:39093,localhost:39094 KAFKA_REPLICATION_FACTOR=3 \
  go run ./cmd/producer -count 1 -topic "$topic" -event-id "evt-$run_id" -order-id "ord-$run_id" -amount-cents 12500
KAFKA_BROKERS=localhost:39092,localhost:39093,localhost:39094 \
  go run ./cmd/consumer -count 1 -topic "$topic" -group "rf3-$run_id"
```

Здесь producer создаёт тему из трёх партиций с replication factor 3 и min ISR 2. Одноузловые defaults приложения равны RF=1; без `KAFKA_REPLICATION_FACTOR=3` команда не создаст требуемую репликацию.

Для управляемой проверки failover выполните отдельный скрипт:

```bash
bash scripts/verify-failover.sh
```

Скрипт поднимает кластер сам и оставляет его запущенным с сохранёнными volumes. Для очистки используйте `cluster down`, а не `cluster reset`, если хотите сохранить лабораторные данные. Остановка двух совмещённых broker/controller узлов потеряет controller quorum, поэтому сценарий не проверяет только порог `min.insync.replicas` при двух потерянных broker-ах.

## Проверки

Unit-тесты и сборка не требуют запущенных контейнеров:

```bash
cd labs/kafka
go test ./...
go test -race ./...
go build ./...
```

Интеграционные тесты используют build tag `integration`, Kafka и PostgreSQL; они создают отдельные имена topic/group и временную DB schema, а затем удаляют их:

```bash
# Выполняйте эту последовательность из корня репозитория.
bash labs/kafka/scripts/lab.sh kraft up --outbox
cd labs/kafka
# Kafka 4.3.1 + PostgreSQL 17.6
go test -tags integration ./...

# Kafka 3.9.2 + тот же уже работающий PostgreSQL
bash scripts/lab.sh zk up
KAFKA_BROKERS=localhost:29092 go test -tags integration ./...
```

Не запускайте эти integration tests против production данных: тестовые topic/group уникальны, а PostgreSQL-тесты создают и удаляют изолированные схемы в настроенной базе. По умолчанию используется только локальная учебная БД.

## Переменные и диагностика

Настройки Go можно переопределить переменными окружения:

| Переменная | Значение по умолчанию | Назначение |
| --- | --- | --- |
| `KAFKA_BROKERS` | `localhost:19092` | Список bootstrap broker-ов через запятую |
| `KAFKA_TOPIC` | `orders` | Основная тема |
| `KAFKA_GROUP` | `orders-lab` | Consumer group |
| `KAFKA_REPLICATION_FACTOR` | `1` | RF новых тем, создаваемых приложением |
| `DATABASE_URL` | Локальная PostgreSQL на `localhost:15432` | Подключение команд outbox и integration tests |
| `KAFKA_LAB_WAIT_TIMEOUT` | `180` секунд | Ожидание Docker healthcheck-ов |

Для Kafka 3.9 укажите `KAFKA_BROKERS=localhost:29092`; для трёхузлового режима используйте broker list `localhost:39092,localhost:39093,localhost:39094` и при создании темы задавайте подходящий replication factor. Внутри контейнеров broker-ы доступны по именам Compose service и listener port `19093`.

Посмотреть состояние выбранного Compose проекта и логи:

```bash
bash scripts/lab.sh kraft status
bash scripts/lab.sh kraft logs
```

Kafka CLI можно запускать внутри контейнера. Например, список тем и состояние групп в KRaft:

```bash
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec broker /opt/kafka/bin/kafka-topics.sh --bootstrap-server broker:19093 --list
docker compose --project-name kafka-handbook-kraft --file compose.kraft.yaml \
  exec broker /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server broker:19093 \
  --describe --group groups-demo-20260929
```

`--describe --group` показывает member assignments, committed offsets и lag для темы и партиции; простой `--list` перечисляет только имена групп и не заменяет этот просмотр. Lag интерпретируйте вместе с концом журнала, состоянием группы и незавершённой обработкой.

## Остановка и сброс

Обычная остановка удаляет контейнеры и сеть, но сохраняет named volumes. Продолжить работу можно повторным `up`:

```bash
bash scripts/lab.sh kraft down
bash scripts/lab.sh kraft up
```

`reset` удаляет данные выбранного учебного Compose project — используйте его только когда состояние больше не нужно:

```bash
bash scripts/lab.sh kraft reset
```

Команды `kraft`, `zk` и `cluster` затрагивают отдельные projects и volumes. `--outbox` поддерживается только режимом `kraft`; для обычных `down`, `reset`, `status` и `logs` скрипт сам добавит profile, если это необходимо.
