# Практика

Задания выполняются в редакторе и Go toolchain. Сайт показывает условия и разборы, но не редактирует код и не запускает его в браузере.

## Как запускать

Команды выполняются из корня репозитория. Для незавершённого стартового решения включите явный тег:

```sh
go test -tags=exercise ./practice/<id>/starter
```

Проверенный вариант запускайте отдельно:

```sh
go test ./practice/<id>/solution
```

Не запускайте `go test ./...`, рассчитывая проверить незавершённые стартовые ответы: они исключены из обычного обхода пакетов. Условия и API хранятся в каноническом `practice/<id>/README.md`, разбор — в `practice/<id>/SOLUTION.md`; страницы сайта подключают эти файлы, поэтому текст не нужно поддерживать второй копией. Подсказки находятся в `<details>` и исключаются из поиска. Ссылки на решения открывают отдельные страницы, которые также исключены из поискового индекса.

## Разбор поведения

- [Алиасинг среза после append](/practice/slice-alias)
- [Изменение среза во время range](/practice/range-mutation)
- [Nil, пустой срез и JSON](/practice/nil-empty-json)
- [Типизированный nil в интерфейсе](/practice/typed-nil)
- [Наборы методов и embedding](/practice/method-sets)
- [Порядок defer](/practice/defer-order)
- [Изменение именованного результата](/practice/named-return)
- [Обход map](/practice/map-iteration)
- [Состояния select](/practice/select-states)
- [Замыкания в цикле](/practice/loop-closures)

## Реализация и отладка

- [Worker pool](/practice/worker-pool)
- [Pipeline](/practice/pipeline)
- [Fan-in](/practice/fan-in)
- [Первый успешный результат](/practice/first-success)
- [Параллельный HTTP-запрос](/practice/url-fetcher)
- [Пакетная обработка через errgroup](/practice/errgroup-batch)
- [HTTP middleware](/practice/http-middleware)
- [Сбор нескольких ошибок](/practice/multi-error)
- [Обобщённое преобразование](/practice/generic-transform)
- [Сборка строки и аллокации](/practice/allocations)

Имена пакетов, типы, гарантии и команды каждого задания описаны на его карточке. Перед решением прочитайте условие целиком; сайт не выполняет код, не отправляет запросы и не сохраняет ход работы.
