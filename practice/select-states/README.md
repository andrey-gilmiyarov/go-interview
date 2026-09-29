# Состояния select

## Задача

ReadyOutcomes — множество допустимых исходов выбора, когда оба buffered-канала готовы; NilBlocks — заблокировалось бы чтение nilChannel; ClosedValue и ClosedOK — результат чтения закрытого исчерпанного канала. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/select-states/contract/example.go

## Контракт и ограничения

<<< @/../practice/select-states/contract/answer.go#answer

Порядок элементов ReadyOutcomes не важен. Один запуск выбирает только один готовый case.

Запустите <code>go test -tags exercise ./practice/select-states/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/select-states/solution</code>.

<details><summary>Подсказка</summary>

Два готовых case допускают оба исхода, но один запуск выбирает только один. Nil-channel case отключён, а закрытый канал читается сразу.

</details>
