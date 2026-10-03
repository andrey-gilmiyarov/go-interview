# Разбор: Восстановление задания после сбоя

## Механизм и решение

Один statement с CTE выбирает FOR UPDATE SKIP LOCKED и обновляет lease/token. Lock освобождается после statement; внешняя работа выполняется вне транзакции. Reclaim увеличивает token. ACK проверяет id, token, срок и done. Старый token не может подтвердить новую попытку.

## Проверка

```sh
./lab.sh run job-queue
go test -tags=exercise ./practice/job-queue/starter
go test -tags=integration ./practice/job-queue/solution
```

Успех означает проверку инвариантов, а не совпадение времени или номера транзакции. При ошибке сначала определите фазу и SQLSTATE; не убирайте assertion ради зелёного результата.

## Senior-усложнение

Token защищает изменение строки очереди, не автоматически внешний сервис. Для эффекта нужны idempotency key или собственная проверка fencing token получателем. Нет гарантии справедливости; для долгих задач потребуется продление lease с тем же token.

## Канонические операции

Эталонный пакет solution вызывает эти же операции, что и эксперимент. Ниже приведён контракт этого задания и общий helper завершения транзакции.

<<< @/../labs/postgres/internal/ops/queue.go

[Вернуться к условию](/postgres/practice/job-queue)

<<< @/../labs/postgres/internal/ops/tx.go
