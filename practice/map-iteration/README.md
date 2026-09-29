# Изменение map во время обхода

## Задача

Заполните три утверждения о гарантиях: OrderGuaranteed — гарантирован ли порядок range; MustVisitInserted — обязана ли итерация посетить запись, добавленную во время range; MustVisitDeletedBeforeReached — будет ли выдан ключ, удалённый до его хода. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/map-iteration/contract/example.go

## Контракт и ограничения

<<< @/../practice/map-iteration/contract/answer.go#answer

Не используйте порядок одного запуска как доказательство гарантии. Начальные ключи, не удаляемые во время обхода, должны встретиться по одному разу.

Запустите <code>go test -tags exercise ./practice/map-iteration/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/map-iteration/solution</code>.

<details><summary>Подсказка</summary>

Сначала отделите гарантии спецификации от результата одного запуска. Допустимый порядок обхода не обязан повторяться.

</details>
