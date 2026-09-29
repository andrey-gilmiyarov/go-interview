# Наборы методов и embedding

## Задача

Предскажите, успешны ли type assertion для T, *T и значения struct{*T} в интерфейс с методом M(). Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/method-sets/contract/example.go

## Контракт и ограничения

<<< @/../practice/method-sets/contract/answer.go#answer

Метод M имеет receiver *T. Метод не вызывается.

Запустите <code>go test -tags exercise ./practice/method-sets/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/method-sets/solution</code>.

<details><summary>Подсказка</summary>

Автоматическое взятие адреса помогает вызвать метод на адресуемом значении. Проверка реализации интерфейса зависит от набора методов типа.

</details>
