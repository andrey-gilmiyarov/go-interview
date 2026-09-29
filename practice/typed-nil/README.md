# Nil-указатель внутри error

## Задача

Предскажите ErrorIsNil для переменной err типа error и PointerIsNil для concrete-переменной pointer. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/typed-nil/contract/example.go

## Контракт и ограничения

<<< @/../practice/typed-nil/contract/answer.go#answer

Не вызывайте Error и не анализируйте вывод форматирования.

Запустите <code>go test -tags exercise ./practice/typed-nil/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/typed-nil/solution</code>.

<details><summary>Подсказка</summary>

Интерфейсное значение содержит динамический тип и динамическое значение. Сравните с nil обе переменные по отдельности.

</details>
