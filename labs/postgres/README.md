# PostgreSQL-лаборатория

Восемь экспериментов для PostgreSQL 17.11 и Go 1.27.1. SQL выполняется на
настоящем сервере; Go-клиент — нативный pgx v5.11.0 с pgxpool. Для чтения сайта
Docker не нужен. Для этой практики нужны Docker Compose и Go.

## Запуск

Из корня репозитория:

```sh
cd labs/postgres
./lab.sh up
./lab.sh ready
./lab.sh status
./lab.sh run stock-race
```

Первый запуск скачивает образ `postgres:17.11-bookworm` и Go-зависимости.
Контейнер публикует только `127.0.0.1:45432`. Если порт занят, задайте
`export POSTGRES_LAB_PORT=45433` перед **всеми** командами, включая Go-тесты.
БД, пользователь и учебный пароль: `postgres_lab`. Это локальная учебная
конфигурация, не production-шаблон секретов/прав доступа.

Проект Compose — `postgres-handbook`, его сеть и volume отдельные от Kafka.
Подключение Go намеренно игнорирует `DATABASE_URL` и `PGHOST`: примеры меняют
только фиксированную локальную БД. Существующий PostgreSQL на 5432 не нужен.

## Сценарии

| ID | Наблюдение |
| --- | --- |
| stock-race | Две продажи последней единицы и атомарное исправление |
| isolation | Statement/transaction snapshot и write skew |
| deadlock | Цикл ожидания и единый порядок блокировок |
| idempotency | Конкурентный повтор и конфликт payload |
| query-plan | EXPLAIN до/после индекса на 100000 строк |
| job-queue | SKIP LOCKED, lease и fencing token |
| long-transaction | Удержание snapshot и VACUUM VERBOSE |
| go-pool | Deadline, отмена и возврат соединения |

```sh
./lab.sh run isolation
./lab.sh run deadlock
./lab.sh run idempotency
./lab.sh run query-plan
./lab.sh run job-queue
./lab.sh run long-transaction
./lab.sh run go-pool
```

Сценарий сам пересоздаёт **свою** схему `lab_<id>` (дефисы заменены на `_`).
Повторный запуск удаляет ваши изменения внутри этой схемы. Не храните там
другие данные. Session advisory lock отклоняет параллельный запуск того же ID.
Разные схемы не зависят друг от друга. Общий deadline — 40 секунд; штатный
прогон быстрее. Барьеры основаны на завершении SQL и `pg_blocking_pids`, а не
на предположении, что goroutine успела выполнить запрос за заданный sleep.

Префикс `PASS <id>` означает проверку заявленных инвариантов. В query-plan
читайте сам план: другой scan или время не означает ошибку. В long-transaction
сравнивайте отчёты VERBOSE и snapshots, а не размер файла.

## Практика и проверка

Условия: [остатки](/postgres/practice/stock-race),
[изоляция](/postgres/practice/isolation), [deadlock](/postgres/practice/deadlock),
[идемпотентность](/postgres/practice/idempotency),
[планы](/postgres/practice/query-plan), [очередь](/postgres/practice/job-queue),
[VACUUM](/postgres/practice/long-transaction), [Go pool](/postgres/practice/go-pool).

```sh
go test ./...
go test -race ./...
go build ./...
./verify.sh
```

Обычные тесты не требуют БД; проверяют retry и конфигурацию. Полная проверка
требует работающей лаборатории и выполняет integration три раза, затем race.
Пакеты выполняются последовательно (`-p=1`), поскольку условия и solution
намеренно используют схемы соответствующих сценариев. Integration без БД
завершается ошибкой, не skip. Упражнения запускаются отдельно:

```sh
go test -tags=exercise ./practice/stock-race/starter
go test -tags=integration ./practice/stock-race/solution
```

Starter собирается, но его проверки ожидаемо падают до решения. Solution —
тонкий вход к каноническим операциям, используемым самой лабораторией;
полный код и объяснение доступны на отдельной странице разбора.

## Дополнительные примеры

`internal/lab/bulk.go` — CopyFrom и batch внутри транзакции. Команда
`go test -tags=integration ./internal/lab -run TestMigrationAndBulkExamples -v`
проверяет его и `examples/migration.sql`. Миграционные команды исполняются
отдельно, в autocommit: CREATE INDEX CONCURRENTLY запрещён в transaction block.
Этот тест готовит свою схему; не запускайте файл против рабочей БД.

## Ручные сессии и диагностика

```sh
./lab.sh psql
```

В каждой сессии явно задайте, например, `SET search_path=lab_isolation,pg_catalog;`.
Автоматический сценарий оставляет данные для исследования; ручные расписания
требуют сброса фикстуры по инструкции задачи. Не запускайте автоматический
reset схемы, пока ваши ручные транзакции открыты.

```sql
SELECT pid, state, xact_start, wait_event_type, wait_event,
       pg_blocking_pids(pid) AS blockers
FROM pg_stat_activity WHERE datname=current_database();
```

Неверный ID, занятый порт и недоступная БД дают ненулевой exit code.
При зависании проверьте открытые ручные транзакции, затем `./lab.sh logs`.
Если sandbox блокирует Docker/socket или локальный TCP, требуется разрешённый
доступ окружения; тест без соединения не является проверкой сценария.

## Остановка

```sh
./lab.sh down
# Только для удаления данных ЭТОЙ лаборатории:
./lab.sh reset
```

`down` сохраняет volume. `reset` удаляет только ресурсы проекта
`postgres-handbook`; после него повторите `up`. Ни одна команда не выполняет
глобальный Docker prune.
