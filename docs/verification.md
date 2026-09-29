# Проверка Go Handbook v1

Полные Go-проверки выполнены после интеграции исходников; последний rework менял документацию и npm preview script, Go-код не менялся.

## Lockfile, контент и сайт

- `npm ci` — PASS: установлены 126 пакетов, проверены 127; npm сообщил о трёх транзитивных advisories (2 moderate, 1 high, наиболее высокий Windows-specific, совместимого исправления нет).
- `npm ls --depth=0` — PASS: VitePress 1.6.4 и Vue 3.5.43.
- `npm run docs:check` — PASS: 8 групп, 42 темы, 20 упражнений; проверены структура и обратные ссылки реестров, canonical practice files и локальные ссылки README/docs.
- `npm run test:site` — PASS: 22/22 теста.
- `git diff --check` и `git diff --cached --check` — PASS; изменения не staged и коммиты не создавались.
- `npm run docs:build` — PASS после article/index правок: VitePress 1.6.4 завершил build за 2.79 s. Первый sandboxed запуск завершился EPERM при записи временного config bundle; повторный запуск с разрешённым filesystem escalation прошёл.
- `node node_modules/vite/bin/vite.js preview site --outDir .vitepress/dist --host 127.0.0.1 --port 4173 --strictPort` — PASS по probe Lead: listener привязан к IPv4 127.0.0.1:4173; вложенный путь `/solutions/slice-alias` вернул HTTP 200. Package script `docs:serve` использует ту же Vite CLI команду; `npm run docs:serve` повторно запущен, listener подтверждён на 127.0.0.1:4173, а /go/ и /go/api/errors проверены в browser.

## Go, TOML и миграция

- `GOTOOLCHAIN=local go version` — PASS: go1.27.1 darwin/arm64.
- `GOPROXY=off GOTOOLCHAIN=local go mod tidy` — PASS; новых зависимостей не добавлялось.
- `GOCACHE=/private/tmp/go-handbook-build GOTOOLCHAIN=local GOPROXY=off go test ./...` — PASS.
- `GOCACHE=/private/tmp/go-handbook-build GOTOOLCHAIN=local GOPROXY=off go test -race ./...` — PASS.
- `GOCACHE=/private/tmp/go-handbook-build GOTOOLCHAIN=local GOPROXY=off go build ./...` — PASS.
- `GOCACHE=/private/tmp/go-handbook-build GOTOOLCHAIN=local GOPROXY=off go vet ./...` — PASS.
- `go test -json -tags=exercise ./practice/.../starter` — ожидаемый exit 1: 20 starter packages содержат 24 намеренно незавершённых проверки; обычные Go checks проходят.
- Python `tomllib` успешно разобрал `.codex/config.toml` и `.codex/agents/luna-worker.toml`; конфигурация не менялась.
- Migration audit — PASS: три migration maps перечисляют все 32 ранее tracked `.go`-файла (14 language, 11 concurrency, 7 backend) и отдельный `test/middleware/main` extensionless Go source.

## Browser QA от Lead

- Прошли маршруты home → Go → slices → practice/slice-alias → явное открытие solution; nested solution reload работает. Все 42 ссылки sidebar доступны, метаданные Go 1.27.1 отображаются.
- Поиск по «срез», «slice», «typed nil», «отмена» релевантен; solution pages не попали в результаты. Все пять моделей корректно проходят scenario/Next/reset; проверены typed nil=false, defer n=0, ожидание полного channel и cooperative worker cancellation/join.
- Details закрыты изначально; Enter раскрывает подсказку и ответ. В размерах 1280×900 и 390×844 для light/dark режимов документ не получил горизонтальную прокрутку, длинный код прокручивается внутри pre. Console warnings/errors пусты.
- Независимый парсер Lead проверил 91 HTML: 0 отсутствующих локальных target/anchor и 0 обязательных удалённых asset URLs. Настоящий screenreader не тестировался.
- После browser pass проверка listener выявила, что старый VitePress preview игнорировал host option и слушал IPv6 wildcard. Тот preview остановлен; исправленная Vite CLI команда выше доказанно слушает только 127.0.0.1. После исправления Lead подтвердил `npm run docs:serve` на IPv4 127.0.0.1:4173, `npm run docs:dev -- --port 4174` на IPv4 127.0.0.1:4174 (dev остановлен после probe), корректное раскрытие canonical ValidateLine snippet и direct reload ошибок, а также корректный intro `/go/`. На свежем search index точечно подтверждено отсутствие `/solutions/` и проверенных уникальных solution-only фраз. Console warnings/errors отсутствуют; preview оставлен запущенным для пользователя.

