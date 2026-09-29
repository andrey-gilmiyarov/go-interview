# Bearer-аутентификация для HTTP handler

## Условие

Реализуйте <code>starter.Handler</code>, который обслуживает один маршрут <code>GET /users/&lt;id&gt;</code>. Для успешного ответа верните JSON-объект с полями <code>id</code> и <code>name</code>. Таблица пользователей и токен передаются конструктору.

~~~go
func Handler(users map[string]string, token string) http.Handler
~~~

<<< @/../practice/http-middleware/example/example.go

## Контракт

Клиент передаёт токен в заголовке <code>Authorization: Bearer &lt;token&gt;</code>. Без точного совпадения верните 401. Неизвестный маршрут или пользователь возвращает 404. Для метода, отличного от GET, верните 405 и заголовок <code>Allow: GET</code>. Проверяйте авторизацию до разбора маршрута и метода. Конструктор копирует map; последующие изменения исходной map не должны менять handler. Пустой token — ошибка конфигурации, конструктор паникует.

Проверка вашей реализации: <code>go test -tags exercise ./practice/http-middleware/starter</code>. Эти tagged-тесты запускают общий набор проверок против кода, который вы меняете в starter. Проверка эталона: <code>go test ./practice/http-middleware/solution</code>; она запускает тот же набор отдельно против независимого reference solution.

<details><summary>Подсказка</summary>

Сначала сравните полный заголовок авторизации. Только затем проверяйте метод и путь.

</details>
