# Пустой и nil-срез в JSON

## Задача

Предскажите равенство каждого среза с nil и строки JSON для nilSlice и emptySlice. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/nil-empty-json/contract/example.go

## Контракт и ограничения

<<< @/../practice/nil-empty-json/contract/answer.go#answer

JSON в этом примере кодируется пакетом encoding/json версии v1.

Запустите <code>go test -tags exercise ./practice/nil-empty-json/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/nil-empty-json/solution</code>.

<details><summary>Подсказка</summary>

len у обоих значений равна нулю. Сравнение с nil и кодирование JSON отвечают на разные вопросы.

</details>
