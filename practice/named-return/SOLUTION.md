# Разбор: named-return

Перед выполнением defer выражение return 4 записывает 4 в именованную переменную result. Отложенная функция изменяет её до окончательного возврата.

Проверка запускает этот Predict и отдельно выполняет канонический пример из contract.

<<< @/../practice/named-return/solution/solution.go

Проверка: <code>go test ./practice/named-return/solution</code>.
