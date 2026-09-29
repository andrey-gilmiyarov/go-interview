# Карта 100 практических пунктов

Эта таблица связывает нумерованные заголовки книги с темами справочника, а не пересказывает содержание книги. Ссылка в первой колонке ведёт к фактическому якорю из проверенного списка заголовков; русские подписи кратко обозначают предмет пункта.

«Включено» отмечено только там, где существующая статья разбирает тот же концепт. Остальные строки — дополнительное чтение до проверки готовой статьи. Сам по себе возраст ссылки не делает материал историческим; отдельные оговорки отмечают семантику, зависящую от версии Go.

| № и исходный якорь | Краткая тема | Связанная тема справочника | Статус и версия |
| --- | --- | --- | --- |
| [#1](https://100go.co/#unintended-variable-shadowing-1) | Тень имени во вложенной области | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#2](https://100go.co/#unnecessary-nested-code-2) | Избыточные уровни ветвления | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#3](https://100go.co/#misusing-init-functions-3) | Побочный эффект инициализации пакета | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#4](https://100go.co/#overusing-getters-and-setters-4) | Методы доступа без нужного контракта | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [#5](https://100go.co/#interface-pollution-5) | Слишком широкий интерфейс | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [#6](https://100go.co/#interface-on-the-producer-side-6) | Интерфейс задаётся слишком рано | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [#7](https://100go.co/#returning-interfaces-7) | Конкретная зависимость за интерфейсом | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [#8](https://100go.co/#any-says-nothing-8) | Смысл any в публичном API | [Обобщения](/go/api/generics) | Дополнительное чтение |
| [#9](https://100go.co/#being-confused-about-when-to-use-generics-9) | Границы применимости обобщений | [Обобщения](/go/api/generics) | Дополнительное чтение |
| [#10](https://100go.co/#not-being-aware-of-the-possible-problems-with-type-embedding-10) | Продвижение метода при встраивании | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [#11](https://100go.co/#not-using-the-functional-options-pattern-11) | Настройка объекта через options | [Функциональные options](/go/patterns/functional-options) | Дополнительное чтение |
| [#12](https://100go.co/#project-misorganization-project-structure-and-package-organization-12) | Структура пакетов проекта | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#13](https://100go.co/#creating-utility-packages-13) | Утилитный пакет без предметной области | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#14](https://100go.co/#ignoring-package-name-collisions-14) | Коллизии импортируемых имён | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение |
| [#15](https://100go.co/#missing-code-documentation-15) | Документация экспортируемого API | [Пакеты и API](/go/api/packages-api), [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение |
| [#16](https://100go.co/#not-using-linters-16) | Автоматическая проверка проекта | [Рабочий цикл и vet](/go/tooling/vet-workflow) | Дополнительное чтение |
| [#17](https://100go.co/#creating-confusion-with-octal-literals-17) | Восьмеричная запись и режим доступа | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение; Сверяйте запись и трактовку режима с текущей спецификацией и целевой ОС. |
| [#18](https://100go.co/#neglecting-integer-overflows-18) | Переполнение машинного целого | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [#19](https://100go.co/#not-understanding-floating-points-19) | Округление двоичных дробей | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [#20](https://100go.co/#not-understanding-slice-length-and-capacity-20) | Длина и запас среза | [Срезы](/go/language/slices) | Включено |
| [#21](https://100go.co/#inefficient-slice-initialization-21) | Выделение места под последовательность | [Срезы](/go/language/slices), [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#22](https://100go.co/#being-confused-about-nil-vs-empty-slice-22) | Nil и пустая коллекция | [Срезы](/go/language/slices), [JSON](/go/stdlib/json) | Включено |
| [#23](https://100go.co/#not-properly-checking-if-a-slice-is-empty-23) | Проверка пустого результата | [Срезы](/go/language/slices) | Дополнительное чтение |
| [#24](https://100go.co/#not-making-slice-copies-correctly-24) | Независимая копия элементов | [Срезы](/go/language/slices) | Включено |
| [#25](https://100go.co/#unexpected-side-effects-using-slice-append-25) | Append и общий базовый массив | [Срезы](/go/language/slices) | Включено |
| [#26](https://100go.co/#slices-and-memory-leaks-26) | Удержание большого массива малым срезом | [Срезы](/go/language/slices), [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#27](https://100go.co/#inefficient-map-initialization-27) | Оценка размера карты | [Карты](/go/language/maps) | Дополнительное чтение |
| [#28](https://100go.co/#maps-and-memory-leaks-28) | Ссылки из карты и время жизни данных | [Карты](/go/language/maps), [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#29](https://100go.co/#comparing-values-incorrectly-29) | Сравнение значений разных типов | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [#30](https://100go.co/#ignoring-that-elements-are-copied-in-range-loops-30) | Копия элемента при range | [Циклы и range](/go/language/control-range) | Включено |
| [#31](https://100go.co/#ignoring-how-arguments-are-evaluated-in-range-loops-channels-and-arrays-31) | Когда вычисляется источник range | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [#32](https://100go.co/#ignoring-the-impacts-of-using-pointer-elements-in-range-loops-32) | Указатели на элементы цикла | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [#33](https://100go.co/#making-wrong-assumptions-during-map-iterations-ordering-and-map-insert-during-iteration-33) | Непредсказуемый обход map | [Карты](/go/language/maps) | Включено |
| [#34](https://100go.co/#ignoring-how-the-break-statement-works-34) | Область действия break | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [#35](https://100go.co/#using-defer-inside-a-loop-35) | Отложенное освобождение в цикле | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [#36](https://100go.co/#not-understanding-the-concept-of-rune-36) | Руна как кодовая точка | [Строки и руны](/go/language/strings-runes) | Включено |
| [#37](https://100go.co/#inaccurate-string-iteration-37) | Байтовый offset и итерация строки | [Строки и руны](/go/language/strings-runes) | Включено |
| [#38](https://100go.co/#misusing-trim-functions-38) | Набор символов для Trim | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [#39](https://100go.co/#under-optimized-strings-concatenation-39) | Сборка строки из частей | [Строки и руны](/go/language/strings-runes), [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#40](https://100go.co/#useless-string-conversions-40) | Преобразование строки и байтов | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [#41](https://100go.co/#substring-and-memory-leaks-41) | Подстрока удерживает исходные байты | [Строки и руны](/go/language/strings-runes), [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#42](https://100go.co/#not-knowing-which-type-of-receiver-to-use-42) | Выбор receiver для метода | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [#43](https://100go.co/#never-using-named-result-parameters-43) | Когда именовать результат | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [#44](https://100go.co/#unintended-side-effects-with-named-result-parameters-44) | Изменение результата при выходе | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [#45](https://100go.co/#returning-a-nil-receiver-45) | Нулевой receiver и контракт типа | [Интерфейсы](/go/api/interfaces), [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [#46](https://100go.co/#using-a-filename-as-a-function-input-46) | Файл как ресурс API | [Ввод-вывод и файлы](/go/stdlib/io-files) | Дополнительное чтение |
| [#47](https://100go.co/#ignoring-how-defer-arguments-and-receivers-are-evaluated-argument-evaluation-pointer-and-value-receivers-47) | Момент вычисления аргумента defer | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [#48](https://100go.co/#panicking-48) | Граница для panic | [defer, panic и recover](/go/language/defer-panic-recover) | Дополнительное чтение |
| [#49](https://100go.co/#ignoring-when-to-wrap-an-error-49) | Контекст при оборачивании ошибки | [Ошибки](/go/api/errors) | Дополнительное чтение |
| [#50](https://100go.co/#comparing-an-error-type-inaccurately-50) | Проверка типа ошибки | [Ошибки](/go/api/errors) | Дополнительное чтение |
| [#51](https://100go.co/#comparing-an-error-value-inaccurately-51) | Сравнение экземпляров ошибок | [Ошибки](/go/api/errors) | Дополнительное чтение |
| [#52](https://100go.co/#handling-an-error-twice-52) | Двойная реакция на одну ошибку | [Ошибки](/go/api/errors) | Дополнительное чтение |
| [#53](https://100go.co/#not-handling-an-error-53) | Потерянная ошибка ветви выполнения | [Ошибки](/go/api/errors) | Дополнительное чтение |
| [#54](https://100go.co/#not-handling-defer-errors-54) | Ошибка во время Close | [Ошибки](/go/api/errors), [Ввод-вывод и файлы](/go/stdlib/io-files) | Дополнительное чтение |
| [#55](https://100go.co/#mixing-up-concurrency-and-parallelism-55) | Конкурентная работа и параллельное исполнение | [Жизненный цикл горутин](/go/concurrency/goroutine-lifecycle), [Планировщик и netpoll](/go/runtime/scheduler-netpoll) | Дополнительное чтение |
| [#56](https://100go.co/#thinking-concurrency-is-always-faster-56) | Накладные расходы конкурентности | [Паттерны конкурентности](/go/patterns/concurrency-patterns) | Дополнительное чтение |
| [#57](https://100go.co/#being-puzzled-about-when-to-use-channels-or-mutexes-57) | Канал или общая блокировка | [Синхронизация](/go/concurrency/sync-primitives), [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [#58](https://100go.co/#not-understanding-race-problems-data-races-vs-race-conditions-and-the-go-memory-model-58) | Гонка данных и гонка бизнес-состояния | [Модель памяти и атомики](/go/concurrency/memory-model-atomic), [Гонки и конкурентные тесты](/go/testing/race-concurrent-tests) | Дополнительное чтение |
| [#59](https://100go.co/#not-understanding-the-concurrency-impacts-of-a-workload-type-59) | Профиль нагрузки и схема конкурентной работы | [Паттерны конкурентности](/go/patterns/concurrency-patterns) | Дополнительное чтение |
| [#60](https://100go.co/#misunderstanding-go-contexts-60) | Передача отмены через context | [context](/go/concurrency/context) | Дополнительное чтение |
| [#61](https://100go.co/#propagating-an-inappropriate-context-61) | Срок жизни context в слоях приложения | [context](/go/concurrency/context) | Дополнительное чтение |
| [#62](https://100go.co/#starting-a-goroutine-without-knowing-when-to-stop-it-62) | Владение остановкой фоновой горутины | [Жизненный цикл горутин](/go/concurrency/goroutine-lifecycle) | Дополнительное чтение |
| [#63](https://100go.co/#not-being-careful-with-goroutines-and-loop-variables-63) | Захват счётчика в цикле | [Циклы и range](/go/language/control-range), [Функции и замыкания](/go/language/functions-closures) | Включено; Go 1.22 изменил переменные, объявленные в заголовке цикла; учитывайте директиву `go` модуля. |
| [#64](https://100go.co/#expecting-a-deterministic-behavior-using-select-and-channels-64) | Выбор среди готовых case | [select](/go/concurrency/select), [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [#65](https://100go.co/#not-using-notification-channels-65) | Событийный канал без payload | [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [#66](https://100go.co/#not-using-nil-channels-66) | Отключение case через nil channel | [select](/go/concurrency/select) | Дополнительное чтение |
| [#67](https://100go.co/#being-puzzled-about-channel-size-67) | Буфер и протокол завершения | [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [#68](https://100go.co/#forgetting-about-possible-side-effects-with-string-formatting-68) | Побочные вызовы при форматировании | [Функции и замыкания](/go/language/functions-closures) | Дополнительное чтение |
| [#69](https://100go.co/#creating-data-races-with-append-69) | Общий append из нескольких горутин | [Срезы](/go/language/slices), [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [#70](https://100go.co/#using-mutexes-inaccurately-with-slices-and-maps-70) | Защита map и среза | [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [#71](https://100go.co/#misusing-syncwaitgroup-71) | Ожидание группы задач | [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [#72](https://100go.co/#forgetting-about-synccond-72) | Ожидание условия с повторной проверкой | [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [#73](https://100go.co/#not-using-errgroup-73) | Отмена группы через errgroup | [golang.org/x/sync](/go/concurrency/x-sync) | Дополнительное чтение |
| [#74](https://100go.co/#copying-a-sync-type-74) | Нельзя копировать sync-примитив | [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [#75](https://100go.co/#providing-a-wrong-time-duration-75) | Единицы time.Duration | [Время и таймеры](/go/stdlib/time-timers) | Дополнительное чтение; Продолжительность задаётся в наносекундах; большие значения требуют проверки переполнения. |
| [#76](https://100go.co/#timeafter-and-memory-leaks-76) | Время жизни одноразового таймера | [Время и таймеры](/go/stdlib/time-timers) | Дополнительное чтение; Старые выводы о памяти и сбросе `time.After` перепроверьте по семантике Go 1.23+. |
| [#77](https://100go.co/#json-handling-common-mistakes-77) | Границы JSON-контракта | [JSON](/go/stdlib/json) | Дополнительное чтение; Явно различайте правила `encoding/json` v1 и v2; упражнение справочника фиксирует v1. |
| [#78](https://100go.co/#common-sql-mistakes-78) | Транзакции и запросы SQL | [database/sql](/go/stdlib/database-sql) | Дополнительное чтение |
| [#79](https://100go.co/#not-closing-transient-resources-http-body-sqlrows-and-osfile-79) | Закрытие HTTP-тела, SQL-строк и файла | [Ввод-вывод и файлы](/go/stdlib/io-files), [HTTP](/go/stdlib/net-http), [database/sql](/go/stdlib/database-sql) | Дополнительное чтение |
| [#80](https://100go.co/#forgetting-the-return-statement-after-replying-to-an-http-request-80) | Выход из handler после HTTP-ответа | [HTTP](/go/stdlib/net-http) | Дополнительное чтение |
| [#81](https://100go.co/#using-the-default-http-client-and-server-81) | Настройка HTTP-клиента и сервера | [HTTP](/go/stdlib/net-http) | Дополнительное чтение |
| [#82](https://100go.co/#not-categorizing-tests-build-tags-environment-variables-and-short-mode-82) | Разделение режимов тестирования | [Unit- и интеграционные тесты](/go/testing/unit-integration) | Дополнительное чтение |
| [#83](https://100go.co/#not-enabling-the-race-flag-83) | Детектор гонок | [Гонки и конкурентные тесты](/go/testing/race-concurrent-tests) | Дополнительное чтение |
| [#84](https://100go.co/#not-using-test-execution-modes-parallel-and-shuffle-84) | Параллельность и shuffle | [Unit- и интеграционные тесты](/go/testing/unit-integration) | Дополнительное чтение |
| [#85](https://100go.co/#not-using-table-driven-tests-85) | Таблица сценариев | [Unit- и интеграционные тесты](/go/testing/unit-integration) | Дополнительное чтение |
| [#86](https://100go.co/#sleeping-in-unit-tests-86) | Ожидание без sleep | [Unit- и интеграционные тесты](/go/testing/unit-integration) | Дополнительное чтение |
| [#87](https://100go.co/#not-dealing-with-the-time-api-efficiently-87) | Уместное использование time API | [Время и таймеры](/go/stdlib/time-timers) | Дополнительное чтение |
| [#88](https://100go.co/#not-using-testing-utility-packages-httptest-and-iotest-88) | Вспомогательные пакеты testing | [Unit- и интеграционные тесты](/go/testing/unit-integration), [HTTP](/go/stdlib/net-http), [Ввод-вывод и файлы](/go/stdlib/io-files) | Дополнительное чтение |
| [#89](https://100go.co/#writing-inaccurate-benchmarks-89) | Точность измерения benchmark | [Бенчмарки](/go/testing/benchmarks) | Дополнительное чтение; Для Go 1.24+ сверяйте `B.Loop` и размещение setup вне измеряемого участка. |
| [#90](https://100go.co/#not-exploring-all-the-go-testing-features-90) | Возможности стандартного testing | [Unit- и интеграционные тесты](/go/testing/unit-integration) | Дополнительное чтение |
| [#91](https://100go.co/#not-understanding-cpu-caches-91) | Локальность CPU и измерения | [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#92](https://100go.co/#writing-concurrent-code-that-leads-to-false-sharing-92) | Конфликт соседних полей кеша | [Аллокации и размещение](/go/runtime/allocations-layout), [Модель памяти и атомики](/go/concurrency/memory-model-atomic) | Дополнительное чтение |
| [#93](https://100go.co/#not-taking-into-account-instruction-level-parallelism-93) | Параллельное исполнение инструкций | [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#94](https://100go.co/#not-being-aware-of-data-alignment-94) | Выравнивание данных платформой | [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение |
| [#95](https://100go.co/#not-understanding-stack-vs-heap-95) | Решение runtime о стеке и куче | [Стек и escape analysis](/go/runtime/stacks-escape) | Дополнительное чтение; Стек или куча — решение компилятора и runtime, а не свойство одного синтаксиса. |
| [#96](https://100go.co/#not-knowing-how-to-reduce-allocations-api-change-compiler-optimizations-and-syncpool-96) | Профиль аллокаций и цена API | [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение; Сравнивайте профиль на целевой версии; не переносите абсолютное число аллокаций. |
| [#97](https://100go.co/#not-relying-on-inlining-97) | Изменчивость inlining | [Аллокации и размещение](/go/runtime/allocations-layout) | Дополнительное чтение; Inlining — оптимизация компилятора, её наличие не является API-гарантией. |
| [#98](https://100go.co/#not-using-go-diagnostics-tooling-98) | Профилирование и диагностика | [pprof и trace](/go/testing/pprof-trace) | Дополнительное чтение |
| [#99](https://100go.co/#not-understanding-how-the-gc-works-99) | Сборка мусора и достижимость | [Сборщик мусора](/go/runtime/garbage-collector) | Дополнительное чтение; GC и runtime-эвристики меняются между выпусками; сверяйте примеры с текущими release notes. |
| [#100](https://100go.co/#not-understanding-the-impacts-of-running-go-in-docker-and-kubernetes-100) | Go под контейнерными CPU-ограничениями | [Планировщик и netpoll](/go/runtime/scheduler-netpoll) | Дополнительное чтение; Современный default `GOMAXPROCS` учитывает доступные CPU и container quota и может обновляться. |

## Дополнение сообщества: fuzzing

Это отдельное дополнение, а не пункт из нумерованной сотни: [официальное руководство Go по fuzz-тестированию](https://go.dev/doc/security/fuzz/) и тема [«Fuzz-тестирование»](/go/testing/fuzzing).

Этот список не подтверждает прочтение книги целиком. Для версии языка и API первичны спецификация, документация стандартной библиотеки и release notes.
