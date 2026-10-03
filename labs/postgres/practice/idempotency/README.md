# Повторное создание заказа

## Условие

Клиент отправил создание заказа, потерял ответ и повторил запрос. Одновременно пришёл ещё один запрос с тем же ключом, но другим количеством.

Реализуйте CreateOrder(ctx,pool,key,product,quantity). Одинаковый запрос возвращает тот же order ID. Другой payload под тем же ключом возвращает ops.ErrConflict. Недостаток stock не оставляет заказ.

## Работа

Из каталога лаборатории после `./lab.sh up`:

```sh
./lab.sh run idempotency
go test -tags=exercise ./practice/idempotency/starter
go test -tags=integration ./practice/idempotency/solution
```

Откройте starter в IDE. Не меняйте контракт функции и общие проверки. Условие использует схему сценария, подготовленную тестом; не запускайте несколько проверок этого ID одновременно.

<details><summary>Подсказка</summary>

Проверка SELECT до INSERT имеет окно гонки. Чем отличается конфликт от ошибки транспортного подтверждения?

</details>

[Глава](/postgres/schema-integrity) · [Отдельный разбор](/postgres/solutions/idempotency)
