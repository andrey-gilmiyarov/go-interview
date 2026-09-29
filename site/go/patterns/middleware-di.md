# Middleware и внедрение зависимостей

## Коротко

В `net/http` обработчик — значение, удовлетворяющее `http.Handler`. Middleware принимает один обработчик и возвращает другой: внешний слой может проверить запрос, добавить наблюдаемость или завершить ответ раньше, чем управление дойдёт до маршрута. Внедрение зависимостей решает отдельный вопрос — откуда handler получает хранилище, часы или клиент. Явный конструктор соединяет зависимости с цепочкой и упрощает тестирование без глобальных переменных.

Порядок обёрток — часть поведения. Для выражения `A(B(C))` запрос проходит сначала через `A`; если `A` вернул ответ без вызова `B`, внутренние слои не запускаются. Ответ проходит обратно через уже выполненные внешние слои только после возврата вложенного вызова.

## Как работает

`http.HandlerFunc` позволяет преобразовать обычную функцию в обработчик, а `ServeMux` выбирает конечный маршрут. Middleware использует ту же форму: функция-обёртка возвращает `http.HandlerFunc` и вызывает следующий handler ровно там, где это разрешено политикой. Слои аутентификации, лимита тела, журнала и recovery могут оборачивать mux. Если журналирование должно фиксировать все запросы, оно располагается снаружи; если внутренние детали должны быть доступны только после проверки прав, этот слой должен быть внутри авторизации.

Зависимость передаётся через конструктор или поле структуры, а не читается из package-level переменной. Интерфейс обычно объявляет потребитель и содержит только нужные ему методы. Конкретное хранилище, fake и адаптер могут тогда использоваться одной реализацией handler без большого интерфейса сервиса — см. [«Пакеты и устойчивые API»](../api/packages-api.md) и [«Интерфейсы»](../api/interfaces.md). В production объект, разделяемый между запросами, должен быть неизменяемым либо синхронизировать собственное состояние.

Начиная с Go 1.22, стандартный `ServeMux` поддерживает шаблоны маршрутов с методом и `Request.PathValue`. Это не меняет внешний контракт middleware: оно по-прежнему видит запрос до маршрутизатора и может завершить его раньше. Ошибки авторизации, отсутствие маршрута, неподдерживаемый метод и ошибка зависимости — разные исходы, каждому нужен осознанный HTTP-статус. Для локальной проверки используйте `httptest`, чтобы не связываться с реальной сетью.

## Пример

`UserStore` описывает только чтение пользователя, нужное handler. Внешний слой `trace` запускается до и после проверки; `authenticate` не вызывает mux при неправильном токене, поэтому поиск пользователя и бизнес-обработчик не выполняются.

~~~go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

type User struct {
	ID   string
	Name string
}

type UserStore interface {
	Find(context.Context, string) (User, bool, error)
}

type memoryUsers map[string]User

func (users memoryUsers) Find(_ context.Context, id string) (User, bool, error) {
	user, ok := users[id]
	return user, ok, nil
}

func authenticate(token string, next http.Handler) http.Handler {
	if token == "" {
		panic("empty token")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("до:", r.URL.Path)
		next.ServeHTTP(w, r)
		fmt.Println("после:", r.URL.Path)
	})
}

func newHandler(store UserStore, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		user, ok, err := store.Find(r.Context(), r.PathValue("id"))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": user.ID, "name": user.Name})
	})
	return trace(authenticate(token, mux))
}

func main() {
	handler := newHandler(memoryUsers{
		"u1": {ID: "u1", Name: "Анна"},
	}, "demo")
	for _, token := range []string{"", "demo"} {
		request := httptest.NewRequest(http.MethodGet, "/users/u1", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		fmt.Println(recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
}
~~~

Локальный запуск выводит статусы `401` и `200`; успешное тело содержит поля `id` и `name`. Внешний `trace` всё равно видит оба запроса, что полезно для метрики отказов. Если политика запрещает даже такую побочную работу до авторизации, перенесите соответствующую обёртку внутрь слоя проверки.

### Составной backend handler

Локально проверяемый пример показывает cookie middleware перед обработчиком маршрута и поведение первого запроса до возврата cookie браузером.

::: details Исходный проверяемый код

<<< @/../examples/backend/user_handlers.go

:::

## Ошибки и ограничения

- Не считайте цепочку middleware нейтральным списком: порядок меняет наблюдаемые заголовки, коды, таймеры и побочные эффекты.
- Не вызывайте следующий handler более одного раза, если протокол и контракт явно не допускают повтор.
- Не закрывайте ответ из middleware до того, как внутренний handler закончил запись; `ResponseWriter` принадлежит вызову `ServeHTTP`.
- Не передавайте изменяемую `map` или общий объект без правил конкурентного доступа. Конструктор упражнения авторизации копирует таблицу пользователей; произвольный адаптер должен обеспечить такую же безопасность самостоятельно.
- Не выдавайте `401` за любую ошибку: отсутствующий пользователь — `404`, запрещённая операция обычно `403`, сбой хранилища — ошибка сервера.
- Не считайте пример с bearer-строкой полноценной схемой аутентификации: production требует TLS, ротацию секрета, лимиты и проверяемую политику идентификации.

Проверяйте и путь, и метод, и заголовки в тестах через `httptest.NewRecorder` или локальный сервер. Отдельно убедитесь, что отказ авторизации не вызвал хранилище. Упражнение [«HTTP middleware с авторизацией»](/practice/http-middleware) фиксирует ещё и копирование входной map, порядок проверки, `Allow: GET` и короткую форму ошибок.

## Самопроверка

<details>
<summary>Почему обёртка может изменить ответ даже без изменения тела handler?</summary>

Она может добавить или перезаписать заголовок, статус, контекст, время, журнал или вовсе не вызвать следующий handler. Senior follow-up: какой порядок гарантирует, что метрика записывает итоговый статус после внутреннего вызова?
</details>

<details>
<summary>Кто объявляет интерфейс для хранилища?</summary>

Обычно handler-потребитель задаёт небольшой интерфейс с необходимой операцией, а пакет хранилища предоставляет конкретный тип. Senior follow-up: какие семантические гарантии, помимо сигнатуры, нужны для безопасной параллельной работы?
</details>

<details>
<summary>Почему авторизация должна завершиться до вызова маршрута?</summary>

Иначе бизнес-обработчик или его зависимость могут произвести эффект до проверки доступа. Внешнее наблюдение при этом может быть намеренно выполнено раньше. Senior follow-up: как тест подтвердит отсутствие вызова хранилища при ошибочном токене?
</details>

## Практика

Соберите цепочку из trace, аутентификации и mux. На тестовом запросе без токена проверьте, что trace увидел путь, ответ равен 401, а fake-хранилище не вызывалось. Затем проверьте 404, неподдерживаемый метод с `Allow: GET` и успешный JSON. Сравните с [HTTP-обработчиком](../stdlib/net-http.md), который владеет протокольной границей, и с упражнением [http-middleware](/practice/http-middleware).

## Источники

- [http.Handler](https://pkg.go.dev/net/http#Handler) — задаёт контракт вызова и время жизни `ResponseWriter` и `Request.Body`.
- [http.HandlerFunc](https://pkg.go.dev/net/http#HandlerFunc) — показывает адаптацию функции к цепочке обработчиков.
- [Go 1.22: ServeMux routing enhancements](https://go.dev/blog/routing-enhancements) — объясняет method-aware patterns и значения пути маршрута.
- [Go Code Review Comments: интерфейсы](https://go.dev/wiki/CodeReviewComments#interfaces) — даёт практическое основание держать интерфейс небольшим и ближе к потребителю.
