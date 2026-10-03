# Проверка PostgreSQL handbook

Дата: 2026-10-03. Статус: реализация принята после self-review и реальных проверок.
План: [PostgreSQL handbook](../plans/2026-10-03-postgresql-handbook.md).

## Среда

- macOS arm64, Go `go1.27.1 darwin/arm64`, Node.js `v22.22.2`.
- Docker client/server 29.1.3. Контейнер: `postgres:17.11-bookworm`.
- SQL `SELECT version()` вернул PostgreSQL 17.11, Debian 17.11-1.pgdg12+2, aarch64.
- pgx v5.11.0 и транзитивные зависимости закреплены в отдельном модуле.
- Для Go-команд использован `GOCACHE=/tmp/postgres-handbook-gocache`:
  стандартный каталог macOS cache недоступен sandbox для записи.
- Доступ к Docker и локальным integration/HTTP-соединениям выполнен через
  разрешённое sandbox escalation. Docker не переустанавливался и не перенастраивался.

## Команды и результаты

Команды сайта и корневого Go-модуля запускались из корня; команды лаборатории —
из `labs/postgres`, если не указано иначе. Exit 0 означает успешное завершение.

| Команда | Точный итог |
| --- | --- |
| `go mod tidy` в лаборатории | Exit 0 после задания writable GOCACHE; только pgx как прямой dependency |
| `npm run test:site` | Exit 0, 43 tests, 43 pass, 0 fail, 0 skip |
| `npm run docs:check` | Exit 0; 8 групп, 42 Go-темы, 20 Go-упражнений, 8 Kafka- и 8 PostgreSQL-глав |
| `npm run docs:build` | Exit 0; предупреждение Vite о chunks >500 kB, без ошибок сборки |
| `docker compose --project-name postgres-handbook --file compose.yaml config --quiet` | Exit 0 |
| `bash -n labs/postgres/lab.sh labs/postgres/verify.sh` из корня | Exit 0 |
| `go test ./...` в лаборатории | Exit 0; без БД, starter/integration не запускаются |
| `go test -race ./...` в лаборатории | Exit 0 |
| `go build ./...` в лаборатории | Exit 0 |
| `go vet ./...` в лаборатории | Exit 0 |
| `./lab.sh up`, `./lab.sh ready` | Exit 0; отдельный контейнер healthy |
| `./verify.sh` | Exit 0, включая обычные тесты, integration с count=3, integration race и build |
| `go test -p=1 -tags=integration -count=3 -timeout=6m ./...` | Exit 0; восемь сценариев, migration/COPY/batch, lock/cancel, четыре solution-контракта |
| `go test -p=1 -race -tags=integration -count=1 -timeout=6m ./...` | Exit 0; без race reports |
| `./lab.sh run stock-race` | Exit 0, PASS stock-race |
| `./lab.sh run isolation` | Exit 0, PASS isolation |
| `./lab.sh run deadlock` | Exit 0, PASS deadlock |
| `./lab.sh run idempotency` | Exit 0, PASS idempotency; ошибочный check-before-insert дополнительно проверен финальным integration-прогоном |
| `./lab.sh run query-plan` | Exit 0, PASS query-plan |
| `./lab.sh run job-queue` | Exit 0, PASS job-queue |
| `./lab.sh run long-transaction` | Exit 0, PASS long-transaction |
| `./lab.sh run go-pool` | Exit 0, PASS go-pool |
| `./lab.sh run missing` | Ожидаемый exit 1, unknown scenario с перечнем допустимых ID |
| `./lab.sh down`, затем `./lab.sh up` | Exit 0; `to_regclass('lab_job_queue.jobs') IS NOT NULL` вернул true: volume сохранён |
| `go test -tags=integration -run TestScenarios/go-pool -count=1 ./internal/lab` при остановленной БД | Ожидаемый exit 1, connection refused на 127.0.0.1:45432, не skip |
| `go test -tags=exercise ./practice/stock-race/starter` | Ожидаемый exit 1: implement me, инвариант покупки не выполнен |
| `go test -tags=exercise ./practice/idempotency/starter` | Ожидаемый exit 1: implement me |
| `go test -tags=exercise ./practice/job-queue/starter` | Ожидаемый exit 1: implement me |
| `go test -tags=exercise ./practice/go-pool/starter` | Ожидаемый exit 1: implement me |
| Корневой `go test ./...` в sandbox | Exit 1: httptest не мог bind локальный порт в examples/backend и url-fetcher/solution |
| Корневой `go test ./...` с разрешением локальной сети | Exit 0, существующие HTTP-тесты проходят |
| Корневой `go build ./...` | Exit 0 |
| `git diff --check`, `git diff --cached --check` | Exit 0 |
| `git status --short` | Изменения только в документации, сайте и новой лаборатории; staging пуст |

## Наблюдения на реальной БД

- Read Committed: чтение 1 → 2; Repeatable Read: 1 → 1.
- Write skew на Repeatable Read: остаток 0. Serializable: одна транзакция
  получила 40001, повтор сохранил остаток 1. Жертва не зафиксирована контрактом.
- Deadlock возвращает 40P01; единый порядок строк позволяет закончить обеим
  транзакциям. Отдельный тест отменяет реально ожидающий запрос.
- Условный UPDATE даёт один заказ из одного остатка; constraint failure при
  последующем INSERT откатывает списание. Конкурентные повторы ключа дают один ID.
- SKIP LOCKED обходит удерживаемую строку; истёкший lease получает новый token;
  старый token не может ACK. Это не гарантия exactly-once внешнего эффекта.
- На 100000 строках наблюдался Seq Scan → Sort → Limit до индекса и
  Index Only Scan → Limit после. Верхний узел: 1924 shared hits до;
  20 hits и 3 reads после. Это измерение данного набора, не обязательный план.
- VACUUM при удерживаемом snapshot: 0 удалённых версий, 1000 dead but not yet
  removable. После COMMIT: 1000 удалено, 0 ещё необходимых dead versions.
  В обоих отчётах осталось 9 страниц: тест не обещает уменьшения файла.
- Исчерпание пула, cancellation, ошибка statement, закрытие rows и rollback
  завершаются возможностью снова получить соединение.

## Найденные и исправленные проблемы

1. Новые registry-тесты сначала падали на отсутствующих Postgres API и обходе
   лабораторных ссылок; после реализации все прошли.
2. Retry-тест сначала не собирался без реализации Retry; затем проверил
   ограниченный повтор 40001/40P01 и единственный вызов при транспортной ошибке.
3. Новый lock/cancellation-тест трижды воспроизвёл 55P03 при немедленном NOWAIT.
   Исходники pgx подтвердили асинхронный cleanup после отмены. Исправлено
   предположение теста: bounded FOR UPDATE проверяет итоговое освобождение
   без произвольного sleep. Пять повторов отдельного теста прошли.
4. Финальная сверка потребовала исполняемого ошибочного варианта идемпотентности.
   Проверка сначала упала с missing reproducible broken variant; добавлена
   гонка двух проверок отсутствия ключа с последующим 23505 у второго INSERT.
   Полный integration/race-прогон после изменения прошёл.

## Проверка сайта и review

В локальном браузере просмотрены обзор, глава изоляции, её таблицы,
условие stock-race и страница решения с Go-кодом. Подсказка раскрывается,
переход к решению работает, версия PostgreSQL отображается. Поиск MVCC
возвращает обычные главы; слово «бессмысленное», присутствующее только в
решении идемпотентности, не даёт результатов. Исключение nested solutions
и подсказок также проверяется тестами.

Self-review выполнен без subagents по правилам репозитория для неподтверждённого
root model. Проверены пять рисков плана: bounded synchronization, неизвестный
COMMIT, stale worker, scope reset, утечка решений в поиск. Reset ограничен
явным Compose project в коде; глобальных prune/удалений нет. Реальный reset
volume не запускался, чтобы сохранить проверенные учебные данные.

Обрыв сети после реального server commit не моделировался: транспортная ошибка
инъецируется в unit-тест политики retry. Нет HA, реплик, нагрузочного benchmark
или доказательства production capacity. Результаты подтверждают учебные
сценарии на указанной версии. TOML не менялся. Staging и commits не выполнялись.

После проверки `./lab.sh down` и `./lab.sh status` завершились с exit 0: контейнер остановлен и удалён, volume сохранён. Локальный preview сайта оставлен на порту 4183.
