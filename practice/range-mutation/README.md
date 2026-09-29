# Изменение среза во время range

## Задача

Предскажите Seen — значения, полученные из range, и Final — x после завершения цикла. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/range-mutation/contract/example.go

## Контракт и ограничения

<<< @/../practice/range-mutation/contract/answer.go#answer

Ответ не включает ёмкость и адрес массива.

Запустите <code>go test -tags exercise ./practice/range-mutation/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/range-mutation/solution</code>.

<details><summary>Подсказка</summary>

range вычисляет диапазон и длину для обхода. Изменение заголовка переменной x не продлевает текущий цикл.

</details>
