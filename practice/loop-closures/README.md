# Замыкание переменной цикла

## Задача

Предскажите результаты функций в Declared и Reused. Каждый срез заполняется вызовом захваченных функций после завершения соответствующего цикла. Верните поля из <code>contract.Answer</code> в <code>starter.Predict()</code>.

<<< @/../practice/loop-closures/contract/example.go

## Контракт и ограничения

<<< @/../practice/loop-closures/contract/answer.go#answer

Учитывайте семантику переменных цикла Go 1.22 и новее.

Запустите <code>go test -tags exercise ./practice/loop-closures/starter</code>, чтобы проверить tagged-заготовку. Тест эталонного ответа запускается отдельно: <code>go test ./practice/loop-closures/solution</code>.

<details><summary>Подсказка</summary>

В Go 1.27 переменная, объявленная в заголовке for, новая для каждой итерации. Переменная, объявленная заранее через var, одна и та же.

</details>
