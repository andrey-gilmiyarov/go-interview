# Ограниченный HTTP fetcher

## Условие

Реализуйте <code>starter.FetchAll</code> для набора URL. Пример принимает client и адреса извне, поэтому его можно использовать с локальным HTTP-сервером.

~~~go
type Result struct {
    URL string
    Status int
    Err error
}
func FetchAll(ctx context.Context, client *http.Client, urls []string, workers int, timeout time.Duration) ([]Result, error)
~~~

<<< @/../practice/url-fetcher/example/example.go

## Контракт

<code>Result</code> содержит исходный URL, HTTP status и ошибку отдельного запроса. Результаты идут в порядке входных URL; ошибка одного запроса хранится в его результате и не отменяет остальные. Одновременно выполняется не больше workers запросов. timeout применяется отдельно к каждому запросу, а не к worker-у. Закройте body каждого полученного response.

Nil client, workers меньше единицы и неположительный timeout дают ошибку. Отмена родителя возвращает <code>nil, ctx.Err()</code> после завершения активных запросов. Тесты используют только локальный сервер и транспорт.

Проверка вашей реализации: <code>go test -tags exercise ./practice/url-fetcher/starter</code>. Она запускает общие проверки с кодом в starter. Эталонная реализация: <code>go test ./practice/url-fetcher/solution</code>.

<details><summary>Подсказка</summary>

Создавайте timeout-контекст после получения каждого URL из очереди и до создания HTTP-запроса.

</details>
