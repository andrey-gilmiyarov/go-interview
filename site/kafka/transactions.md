# Идемпотентность и транзакции Kafka

## Коротко

Idempotent producer защищает Kafka-журнал от дублей, вызванных повтором протокольной отправки тем же producer state. Это не дедупликация прикладного event_id и не транзакция с PostgreSQL или HTTP. Транзакция Kafka может атомарно объединить записи в Kafka и offsets входной группы; внешний эффект требует собственного атомарного механизма.

## Как работает

При сетевом таймауте producer может не знать, успел ли broker записать batch до потери ответа. Идемпотентный producer сообщает broker идентификатор producer/sequence, чтобы повтор того же протокольного batch не стал второй копией записи. Это покрывает retry в контексте producer identity, а не два разных вызова приложения. Если сервис повторно отправил evt-01 после перезапуска как новую операцию, либо пересоздал бизнес-событие с другим ID, Kafka не знает, что бизнес-смысл совпадает. Consumer всё равно проектируют на повторную доставку.

Транзакционный producer может писать в несколько Kafka partitions/topics и включать в ту же транзакцию offsets обработанных входных partitions. Алгоритм consume-transform-produce: получить входную запись, рассчитать выход, добавить output records, добавить offsets текущей обработки, затем commit Kafka transaction. Abort оставляет output невидимым для читателя с read_committed и не продвигает offsets транзакционно. Для правильного восстановления consumer снова читает вход с последней зафиксированной позиции; схема кода учитывает ownership группы и корректное завершение транзакции.

read_committed возвращает committed transactional records и нетранзакционные записи. Чтение ограничивается last stable offset, поэтому открытая транзакция может задержать видимость более поздних записей этой партиции. read_uncommitted может вернуть записи, которые позже будут aborted. Это свойство чтения, а не обещание, что произвольная база данных увидит все или только committed события.

Граница exactly-once здесь — обработка Kafka→Kafka с output и input position, которые входят в одну Kafka transaction. Если внутри цикла выполнить HTTP-вызов или записать PostgreSQL, Kafka не может отменить его при abort; включение read_committed этого не меняет. В Kafka 4.0 появился усиленный server-side transaction protocol, но broker/client version negotiation всё равно остаётся отдельным фактором. Конкретный producer API в этой лаборатории — franz-go v1.22.1; producer properties из Java документации не являются списком Go options.

## Пример

Посмотрите транзакционный consumer-transform-producer в лаборатории:

<<< @/../labs/kafka/internal/kafka/consume.go#consume-transactionally

Важные границы показаны вместе: input topic и output topic передаются приложению, отказ после output должен привести к abort, а читающий output consumer использует read_committed. Проверьте defaults и failpoint в исходнике, а полный запуск — в [README лаборатории](/kafka/lab).

## Ошибки и ограничения

- Идемпотентная протокольная повторная отправка не заменяет event_id и бизнес-дедупликацию.
- Новый producer identity или новая прикладная отправка — не тот же retry, который broker обязан распознать как duplicate.
- Транзакция охватывает только ресурсы Kafka, участвующие в протоколе; внешний DB update или HTTP side effect не откатывается.
- Consumer с read_uncommitted может наблюдать output отменённой транзакции.
- read_committed может ждать завершения более ранней незавершённой транзакции, прежде чем выдать более поздний offset.
- Гарантии зависят от broker, client, выбранных options и корректного включения offsets в transaction.

## Самопроверка

<details>
<summary>Уберёт ли Kafka дубль, если приложение после перезапуска заново создаст запись с тем же event_id?</summary>

Нет. Идемпотентность Kafka относится к повтору протокольной отправки в рамках producer state. event_id — поле payload и не становится broker-side уникальным ключом. Для прикладной дедупликации нужен уникальный ключ у получателя или иная явно спроектированная операция.

</details>

<details>
<summary>Что именно делает транзакцию consume-transform-produce exactly-once в пределах Kafka?</summary>

Output records и offsets прочитанного входа атомарно фиксируются одной Kafka transaction. Если она abort-ится, read_committed не показывает output и транзакционный commit входной позиции не завершён; повторная попытка читает вход снова. Внешняя система в эту гарантию не входит без собственной координации.

</details>

## Практика

В [лаборатории](/kafka/lab) используйте сценарий успешной транзакции и сценарий failpoint после отправки output, затем читайте результат в read_committed. Лаборатория также показывает отменённую транзакцию; не используйте её output как подтверждение, что внешний эффект можно откатить.

<details>
<summary>Решение: выходная запись отправлена, затем приложение abort-ило транзакцию. Что увидят два consumer с разной изоляцией?</summary>

read_committed не выдаст запись отменённой транзакции и может удерживать чтение после незавершённого transaction boundary. read_uncommitted способен увидеть aborted запись. В обоих случаях это не отменит никакую PostgreSQL-строку или HTTP request, уже выполненные приложением вне Kafka transaction.

</details>

## Источники

- [Kafka 4.3: Design — доставка, producer idempotence и transactions](https://kafka.apache.org/43/design/design/)
- [Kafka 4.3: producer configs](https://kafka.apache.org/43/configuration/producer-configs/)
- [Kafka 4.3: consumer configs — isolation.level](https://kafka.apache.org/43/configuration/consumer-configs/)
- [franz-go v1.22.1: kgo API](https://pkg.go.dev/github.com/twmb/franz-go@v1.22.1/pkg/kgo)
