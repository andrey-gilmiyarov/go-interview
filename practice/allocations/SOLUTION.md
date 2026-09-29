# Разбор: allocations

Join заранее считает размер результата, включая разделители, выделяет один буфер нужной длины и копирует в него части и запятые. Это сохраняет поведение strings.Join для пустых и Unicode-строк, не создавая промежуточную строку на каждом шаге.

Общий benchmark сравнивает текущую функцию Join с наивной конкатенацией на фиксированном наборе строк. Он доступен и в tagged starter test, чтобы можно было измерить собственную реализацию. ReportAllocs показывает измерения конкретного запуска; они зависят от архитектуры и версии Go и не являются гарантией API. Сравнение array-of-structs и struct-of-slices опубликовано в теме [layout данных](/go/runtime/allocations-layout).

<<< @/../practice/allocations/solution/solution.go

Проверка эталона: <code>go test ./practice/allocations/solution</code>. Benchmark: <code>go test -run '^$' -bench . -benchmem ./practice/allocations/solution</code>.
