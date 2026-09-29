# Ошибки для нескольких значений

## Условие

Реализуйте <code>starter.Validate</code>. Функция проверяет весь список и сообщает об отрицательных значениях, сохраняя индекс каждого нарушения.

~~~go
func Validate(values []int) error
~~~

<<< @/../practice/multi-error/example/example.go

## Контракт

Для каждого отрицательного элемента создайте <code>*ItemError{Index, Value}</code>, который оборачивает экспортированный sentinel <code>ErrNegative</code>. Верните все нарушения через <code>errors.Join</code>. Для допустимого списка верните настоящий nil, а не интерфейс с nil concrete value. Проверяющий код должен находить sentinel через <code>errors.Is</code>, а элемент ошибки через <code>errors.As</code>.

Проверка вашей реализации: <code>go test -tags exercise ./practice/multi-error/starter</code>. Эти tagged-тесты запускают общий набор проверок против starter, который вы меняете. Проверка эталона: <code>go test ./practice/multi-error/solution</code>; она запускает тот же набор отдельно против независимого reference solution.

<details><summary>Подсказка</summary>

Пустой срез ошибок передайте в <code>errors.Join</code>: он возвращает nil.

</details>
