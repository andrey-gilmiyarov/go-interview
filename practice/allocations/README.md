# Соединение строк и аллокации

## Условие

Реализуйте <code>starter.Join</code>: соедините элементы через запятую, сохранив семантику <code>strings.Join</code>.

~~~go
func Join(parts []string) string
~~~

<<< @/../practice/allocations/example/example.go

## Контракт

Пустой или nil-вход даёт пустую строку. Пустые элементы и Unicode-текст сохраняются. Не меняйте входной срез. Эталон сравнивается с <code>strings.Join(parts, ",")</code> на таблице значений. В benchmark сравните вашу реализацию с простым повторным concatenation и включите <code>ReportAllocs</code>; тесты не фиксируют число байтов или абсолютное время.

Проверка вашей реализации: <code>go test -tags exercise ./practice/allocations/starter</code>. Tagged-тесты запускают общий набор проверок против starter, который вы меняете. Проверка эталона: <code>go test ./practice/allocations/solution</code>; она запускает тот же набор отдельно против независимого reference solution.

Запустить benchmark для starter: <code>go test -tags exercise -run '^$' -bench . -benchmem ./practice/allocations/starter</code>. Для эталона используйте ту же команду без <code>-tags exercise</code>, заменив пакет на <code>./practice/allocations/solution</code>. Смотрите на результат своего запуска: проверка не задаёт порог скорости или числа аллокаций.

<details><summary>Подсказка</summary>

Посчитайте итоговую длину до выделения буфера.

</details>
