# Transactional outbox

## Коротко

Transactional outbox атомарно сохраняет изменение заказа и намерение опубликовать событие в одной PostgreSQL transaction. Polling publisher затем отправляет ожидающую строку в Kafka и после подтверждения помечает её опубликованной; падение между send и mark создаёт повтор, поэтому доставка остаётся at-least-once. Outbox устраняет потерю между двумя независимыми записями, но не делает PostgreSQL и Kafka одной распределённой транзакцией.

## Как работает

Прямой dual write выглядит так: сервис commit-ит заказ в PostgreSQL, затем отправляет Kafka message. Если процесс упадёт между шагами, состояние заказа уже существует, а события нет. Обратный порядок тоже небезопасен: запись может оказаться в Kafka, хотя заказ откатился. В transactional outbox сервис сохраняет заказ и запись outbox атомарно внутри PostgreSQL; отдельный publisher выбирает только committed строки.

Polling lifecycle состоит из двух системных шагов. Publisher читает ожидающую строку, формирует record с key заказа и стабильным event_id, ждёт успешного подтверждения Kafka, затем помечает строку опубликованной. Если процесс упал до Kafka send, строка остаётся ожидающей. Если Kafka подтвердила send, но процесс упал перед отметкой в БД, следующий polling может отправить запись второй раз. Ставить published до send опаснее: после сбоя между update и сетевым запросом событие потеряется. Значит, outbox обычно выбирает дубликат с дедупликацией на стороне получателя.

Текущая лаборатория использует одного publisher. `PublishPending` выбирает ограниченный batch через `FOR UPDATE`, удерживает PostgreSQL transaction и блокировки строк во время Kafka send, а на каждую отправку задаёт deadline 10 секунд; затем одной DB-транзакцией сохраняет markers. Это намеренный простой учебный компромисс: ограниченный batch и timeout сдерживают длительность блокировки, но полное время транзакции зависит от числа записей и задержек Kafka. Для production нужно отдельно проектировать lease/claim и recovery, размер batch, timeout и конкуренцию.

Глава не утверждает, что текущая лаборатория запускает несколько publisher-ов или проверяет их порядок. При горизонтальном масштабировании наивные независимые poller-ы могут отправить события одного заказа не в порядке создания. Kafka key упорядочит только фактическую последовательность append в одной партиции; последовательность выдачи из БД и отправки до Kafka нужно защищать отдельно, например sequence/version заказа и согласованным механизмом публикации.

CDC — альтернативный механизм чтения outbox, а не переключение одного флага в polling коде. Debezium PostgreSQL connector читает logical decoding/WAL через replication slot; slot удерживает нужные WAL segments во время простоя, поэтому нужно следить за диском и восстановлением. Outbox Event Router ожидает INSERT-события outbox определённого формата и имеет правила для UPDATE/DELETE, поэтому текущая polling-схема с обновлением статуса публикации потребовала бы адаптации таблицы, connector/router config, cutover, replay и дедупликации. В этой лаборатории Debezium и Kafka Connect не запускаются: CDC только обозначает будущее направление.

## Пример

`CreateOrder` в лаборатории вставляет заказ и outbox row одной PostgreSQL transaction:

<<< @/../labs/kafka/internal/store/store.go#create-order

`PublishPending` блокирует ограниченную выборку через `FOR UPDATE`, отправляет records с индивидуальным deadline и фиксирует published markers одной DB transaction. Поэтому строковые блокировки остаются заняты во время сетевых отправок; это учебный компромисс одного poller-а, а не готовая схема горизонтального масштабирования:

<<< @/../labs/kafka/internal/store/store.go#publish-pending

Форма таблиц, переменные подключения, публикация и проверка строк описаны в [README лаборатории](/kafka/lab). Именно он, а не эта глава, задаёт точные CLI arguments и команды Compose.

## Ошибки и ограничения

- Запись заказа и outbox row в разных транзакциях возвращает исходное окно dual write.
- Удалять outbox row сразу после отправки опасно для аудита и replay; политику очистки и срок хранения выбирает приложение.
- Published marker обновляется после подтверждения Kafka, поэтому duplicate send между этим подтверждением и DB update ожидаем.
- Одинаковый key обеспечивает partitioning только согласно client partitioner; он не сериализует несколько poller-ов до обращения к Kafka.
- Poller удерживает row locks и транзакцию во время сетевого send; ограниченные batch и timeout задают trade-off, а не устраняют его.
- Параллельная публикация и сохранение порядка не являются проверяемыми функциями текущей лаборатории; для неё используется один publisher.
- Debezium требует эксплуатации Kafka Connect, logical replication, slots и WAL backlog; polling implementation нельзя считать CDC-готовой автоматически.

## Самопроверка

<details>
<summary>В каком окне outbox publisher может отправить повторное событие?</summary>

После того, как Kafka уже подтвердила запись, и до того, как PostgreSQL зафиксировал published marker. После рестарта строка выглядит неотправленной и будет отправлена ещё раз; downstream inbox по стабильному event_id закрывает это окно для DB-эффекта.

</details>

<details>
<summary>Почему Kafka key не гарантирует порядок outbox-событий одного заказа при нескольких poller-ах?</summary>

Key направляет отправленные записи в одну партицию, где сохраняется порядок append. Но конкурентные poller-ы могут получить строки в исходном DB-порядке и вызвать send в другом порядке. Сериализацию нужно обеспечить на уровне выборки/публикации или проверять order sequence downstream.

</details>

## Практика

В [лаборатории](/kafka/lab) создайте заказ, опубликуйте outbox row и повторите publisher после заданного failpoint после подтверждения Kafka и до отметки PostgreSQL. Проверьте, что Kafka допускает duplicate, а consumer/inbox применяет бизнес-эффект однократно.

<details>
<summary>Решение: в Kafka два одинаковых event_id, а в таблице заказа один бизнес-эффект. Это поломка?</summary>

Это ожидаемое сочетание at-least-once публикации outbox и идемпотентного потребителя. Убедитесь, что строки Kafka действительно повторяют одно событие с одинаковым event_id, а inbox уникален по этому ID и меняется в одной DB transaction с заказом. Затем проверьте состояние published marker и причину повторной попытки.

</details>

## Источники

- [Kafka 4.3: Design — delivery semantics](https://kafka.apache.org/43/design/design/)
- [Kafka 3.9: Design](https://kafka.apache.org/39/design/design/)
- [PostgreSQL 17: транзакции](https://www.postgresql.org/docs/17/tutorial-transactions.html)
- [Debezium: PostgreSQL connector, logical decoding и replication slots](https://debezium.io/documentation/reference/stable/connectors/postgresql.html)
- [Debezium: Outbox Event Router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html)
- [franz-go v1.22.1: kgo API](https://pkg.go.dev/github.com/twmb/franz-go@v1.22.1/pkg/kgo)
