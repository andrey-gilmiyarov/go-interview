# PostgreSQL handbook — план реализации

Статус: завершён и проверен. Спецификация утверждена пользователем 2026-10-03.

Основа: [спецификация](../specs/2026-10-03-postgresql-handbook-design.md).

Цель: восемь глав PostgreSQL, восемь отдельных задач и воспроизводимая лаборатория.
Стек: PostgreSQL 17.11-bookworm, Go 1.27.1, pgx v5.11.0, существующий VitePress.
Работа в основном checkout, без staging/commits и обновления старых зависимостей.

## Этапы

- [x] 1. Лаборатория: отдельный Go-модуль, Compose project postgres-handbook,
  БД postgres_lab, loopback 45432 с POSTGRES_LAB_PORT; lab.sh
  up/ready/status/logs/down/reset/psql/run; verify.sh. Свои схемы сценариев,
  advisory lock, readiness/deadline и cleanup с отдельным context.
- [x] 2. stock-race, isolation, deadlock, idempotency. Ошибочные расписания и
  исправления, SQLSTATE, прогноз видимости, атомарность заказа/остатка, конфликт
  payload. Go starter/solution для stock-race и idempotency; SQL для остальных.
- [x] 3. query-plan, job-queue, long-transaction, go-pool. Реальные планы без
  порогов времени, lease/token и stale worker, snapshot/VACUUM, pool cancellation.
  Go starter/solution для job-queue и go-pool. Ограниченный retry, тест
  неизвестного COMMIT через внедряемую ошибку. CopyFrom, batch и миграции.
- [x] 4. Восемь глав из спецификации с семью разделами, минимум тремя вопросами,
  первичными источниками, таблицами расписаний, отдельными условиями/решениями.
  database/sql сопоставляется с pgx, новые интерактивные компоненты не нужны.
- [x] 5. PostgreSQL-реестр, loadPostgresRegistry/buildPostgresSidebar,
  stats.postgresTopics, strict/partial проверки, обход labs/postgres,
  навигация /postgres и практика/решения, исключение ответов из поиска.
  Обновление главной, каталога, README, roadmap и авторской памятки.
- [x] 6. Верификация и review: сайт, реальные сценарии PostgreSQL, race,
  регрессии корневого Go-модуля, ссылки/diff/status и визуальная проверка сайта.

## Проверки

Сайт: npm run test:site; npm run docs:check; npm run docs:build.
Лаборатория: docker compose config --quiet; go test ./...; go test -race ./...;
go build ./...; ./lab.sh up; ./verify.sh (integration -count=1 и race).
Корень: go test ./...; go build ./...; git diff --check;
git diff --cached --check; git status --short.
Starter включается только тегом exercise. Integration без БД обязан завершаться
ошибкой. Каждый сценарий запускается отдельно и три раза последовательно.

## Review focus

Зависшие барьеры; слепой retry неизвестного COMMIT; stale worker acknowledgement;
reset чужой БД; раскрытие решений. Сохранять 42 Go-темы, 20 упражнений и 8 Kafka-глав.

## Решения и журнал

- Root model не подтверждён: самостоятельное выполнение и self-review согласно AGENTS.md.
- Docker daemon доступен после разрешённого sandbox escalation; реальная проверка возможна.
- Все новые прямые Go-зависимости ограничены утверждённым pgx, транзитивные закрепляются.
- Итоговые команды и результаты: [отчёт](../reports/postgresql-verification.md).

- Ruling: cancellation cleanup проверяется bounded acquisition, не немедленным NOWAIT: pgx асинхронно закрывает соединение после отмены; это подтверждено pgconn.asyncClose/CleanupDone и воспроизводимым тестом.

- Итог: восемь сценариев прошли реальные integration (3 повтора) и race; 43 site tests проходят, build/link checks и корневые Go-проверки успешны.
