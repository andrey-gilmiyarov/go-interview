# Разбор: nil-empty-json

У nil-среза и пустого непустого среза одинаковая длина, но только nil-срез равен nil. encoding/json v1 кодирует их как null и [].

Проверка запускает этот Predict и отдельно выполняет канонический пример из contract.

<<< @/../practice/nil-empty-json/solution/solution.go

Проверка: <code>go test ./practice/nil-empty-json/solution</code>.
