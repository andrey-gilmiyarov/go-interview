# Порядок defer и момент вычисления

## Задача

Предскажите Events — значения, записанные функцией log по порядку фактического выполнения. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/defer-order/contract/example.go

## Контракт и ограничения

<<< @/../practice/defer-order/contract/answer.go#answer

Первый вызов defer передаёт аргумент log(x); второй регистрирует замыкание, которое обращается к x.

Запустите <code>go test -tags exercise ./practice/defer-order/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/defer-order/solution</code>.

<details><summary>Подсказка</summary>

Аргументы отложенного вызова вычисляются при регистрации. Отложенная функция-замыкание читает переменную при выполнении.

</details>
