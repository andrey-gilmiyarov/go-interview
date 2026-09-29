# Алиасинг среза после append

## Задача

Предскажите два снимка именно исходного массива a: Shared берётся сразу после первого append, а Detached — после второго append и записи в s[0]. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/slice-alias/contract/example.go

## Контракт и ограничения

<<< @/../practice/slice-alias/contract/answer.go#answer

Shared и Detached — копии a, а не возвращённое значение s. Не предсказывайте ёмкость или адрес массива.

Запустите <code>go test -tags exercise ./practice/slice-alias/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/slice-alias/solution</code>.

<details><summary>Подсказка</summary>

Срез хранит заголовок, а append возвращает новый заголовок. Для второго append отдельно оцените длину и доступный хвост исходного массива.

</details>
