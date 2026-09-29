# Ограниченный пакет задач через errgroup

## Условие

Реализуйте <code>starter.Run</code> для запуска функций, которым передаётся контекст группы.

~~~go
func Run(ctx context.Context, jobs []func(context.Context) error, limit int) error
~~~

<<< @/../practice/errgroup-batch/example/example.go

## Контракт

Используйте <code>errgroup.WithContext</code> и <code>SetLimit</code>. Значение <code>limit</code> должно быть положительным; иначе Run возвращает ошибку. Положительный limit ограничивает число активных jobs. Ошибка job отменяет контексты siblings; jobs, которые ещё не начали callback к моменту отмены, пропускаются. Родительская отмена возвращается и для пустого списка. Перед возвратом функция ждёт завершения запущенных jobs; callbacks должны учитывать контекст.

Проверка вашей реализации: <code>go test -tags exercise ./practice/errgroup-batch/starter</code>. Она запускает общие проверки с кодом в starter. Эталонная реализация: <code>go test ./practice/errgroup-batch/solution</code>.

<details><summary>Подсказка</summary>

Проверяйте контекст внутри функции, переданной в группу: SetLimit может удерживать отправителя, пока освобождается слот.

</details>
