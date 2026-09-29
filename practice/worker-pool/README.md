# Ограниченный worker pool

## Условие

Реализуйте <code>starter.Map</code>: примените переданную функцию к каждому числу и верните результаты в исходном порядке. Максимум одновременно выполняются <code>workers</code> вызовов. Функция получает общий рабочий контекст и должна завершаться после его отмены.

~~~go
func Map(ctx context.Context, values []int, workers int, fn func(context.Context, int) (int, error)) ([]int, error)
~~~

<<< @/../practice/worker-pool/example/example.go

## Контракт

Число workers должно быть положительным. Пустой вход без отменённого контекста даёт пустой результат. При первой ошибке callback или отмене контекста Map возвращает <code>nil, error</code> после завершения всех уже запущенных вызовов функции. Callback обязан завершаться после отмены переданного контекста.

Проверка вашей реализации: <code>go test -tags exercise ./practice/worker-pool/starter</code>. Она запускает общие проверки с кодом в starter. Эталонная реализация: <code>go test ./practice/worker-pool/solution</code>.

<details><summary>Подсказка</summary>

Передавайте индекс вместе с заданием, чтобы собрать упорядоченный результат независимо от того, какой worker закончил раньше.

</details>
