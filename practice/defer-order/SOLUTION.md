# Разбор: defer-order

Первый аргумент defer вычислен при регистрации, когда x равен 1. Замыкание читает x позднее. Defer выполняются в обратном порядке регистрации.

Проверка запускает этот Predict и отдельно выполняет канонический пример из contract.

<<< @/../practice/defer-order/solution/solution.go

Проверка: <code>go test ./practice/defer-order/solution</code>.
