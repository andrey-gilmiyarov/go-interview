# Изменение именованного результата

## Задача

Предскажите Result — значение, которое функция вернёт вызывающей стороне. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/named-return/contract/example.go

## Контракт и ограничения

<<< @/../practice/named-return/contract/answer.go#answer

У функции именованный результат result; отложенная функция меняет именно его.

Запустите <code>go test -tags exercise ./practice/named-return/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/named-return/solution</code>.

<details><summary>Подсказка</summary>

return сначала присваивает выражение именованной переменной результата, затем выполняет defer и только после этого возвращает результат.

</details>
