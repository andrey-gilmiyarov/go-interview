# Преобразование среза любого типа

## Условие

Реализуйте <code>starter.Transform</code> как generic-функцию. Она применяет callback к каждому элементу и собирает значения результата.

~~~go
func Transform[T any, R any](values []T, fn func(T) R) []R
~~~

<<< @/../practice/generic-transform/example/example.go

## Контракт

Сохраните длину и порядок. Если входной срез nil, верните nil. Для ненулевого пустого среза верните пустой ненулевой результат. Поддержите разные типы входа и выхода. Ограничение <code>any</code> подходит, потому что функция только передаёт значения callback и не требует сравнения или операций map key.

Проверка вашей реализации: <code>go test -tags exercise ./practice/generic-transform/starter</code>. Tagged-тесты запускают общий набор проверок против starter, который вы меняете. Проверка эталона: <code>go test ./practice/generic-transform/solution</code>; она запускает тот же набор отдельно против независимого reference solution.

<details><summary>Подсказка</summary>

Выделите результат длиной во входной срез и заполните соответствующие индексы.

</details>
