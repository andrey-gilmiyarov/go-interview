# Разбор: typed-nil

В error после преобразования хранится динамический тип *MyError, поэтому интерфейсное значение не nil. Значение указателя при этом остаётся nil.

Проверка запускает этот Predict и отдельно выполняет канонический пример из contract.

<<< @/../practice/typed-nil/solution/solution.go

Проверка: <code>go test ./practice/typed-nil/solution</code>.
