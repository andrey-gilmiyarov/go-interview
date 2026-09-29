# Первый успешный результат

## Условие

Реализуйте <code>starter.First</code> для набора независимых попыток, например поиска свободного такси у нескольких служб.

~~~go
var ErrNoJobs error
func First(ctx context.Context, jobs ...func(context.Context) (string, error)) (string, error)
~~~

<<< @/../practice/first-success/example/example.go

## Контракт

Верните первый замеченный успешный результат. Его появление отменяет контексты siblings; функция должна дождаться всех уже запущенных функций до возврата. Если работ нет, верните <code>ErrNoJobs</code>. Если все callbacks завершились ошибками, верните ошибку, объединяющую их через <code>errors.Join</code>. Если родительский контекст отменён до принятия успеха, верните его ошибку. Каждая функция обязана завершаться после отмены контекста.

Проверка вашей реализации: <code>go test -tags exercise ./practice/first-success/starter</code>. Она запускает общие проверки с кодом в starter. Эталонная реализация: <code>go test ./practice/first-success/solution</code>.

<details><summary>Подсказка</summary>

Выберите ровно один успех, а затем дождитесь завершения callbacks, уже получивших контекст задачи.

</details>
