# Корректное завершение сервиса

## Коротко

Graceful shutdown — это управляемый переход от приёма новых работ к завершению уже принятых. Для HTTP-сервера сначала останавливают admission, затем ограниченно ждут активные обработчики, присоединяют цикл `Serve` и фоновые задачи, и только после этого закрывают зависимости, которыми те пользуются. Сигнал отменяет контекст остановки, но этот уже отменённый контекст нельзя передавать в drain.

Deadline не останавливает произвольные горутины: после его истечения обработчики и WebSocket ещё могут использовать ресурсы. Политика force-close должна сохранить ошибку, проверить собственные join и оставить внешний предел supervisor-у.

## Как работает

Корневой сигнал обычно получают через `signal.NotifyContext` и освобождают его `stop`-функцией после завершения остановки. При сигнале `http.Server.Shutdown` закрывает listeners, закрывает простаивающие соединения и ждёт, пока активные обработчики закончат. Сам `Serve` возвращает `http.ErrServerClosed`; главный поток должен прочитать этот результат, а не выйти сразу после вызова `Shutdown`. `Shutdown` не закрывает и не ждёт соединения, извлечённые через `Hijacker`, поэтому WebSocket и похожий долгоживущий протокол требуют собственного реестра, уведомления и join.

Drain получает новый bounded context от `context.Background()`. Если контекст сигнала уже отменён, производный от него `WithTimeout` немедленно истечёт и остановка превратится в force close. Ограниченный drain защищает deploy и процесс от бесконечного ожидания, а не гарантирует успешное завершение за любое заданное время.

Если handler ставит задачу в очередь, остановите HTTP admission прежде, чем закрывать очередь. После возврата handlers закройте её силами владельца, дайте workers обработать остаток, присоедините их и закройте общие ресурсы. Отдельно решите, должна ли активная фоновая задача завершиться, отмениться или быть отброшена по deadline.

При таймауте `Shutdown` вызов `Server.Close` закрывает сетевые соединения, но не принудительно прерывает произвольный код handler и не ждёт все такие горутины. Поэтому пример ниже закрывает зависимость только после успешного drain. Если нужен force close, учтите собственные горутины и не закрывайте их общую зависимость, пока ваш join не подтвердил завершение. Принудительное завершение процесса в крайнем случае — политика supervisor-а, а не гарантия `context`.

## Пример

Это компилируемый фрагмент пакета: приложение передаёт ему уже созданный listener и функцию закрытия зависимостей. При штатном drain функция закрывает listener, присоединяет Serve и только затем вызывает closeResources. После ошибки drain она закрывает сетевые соединения, но не закрывает зависимость, которой handler ещё может пользоваться.

~~~go
package lifecycle

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

const drainTimeout = 8 * time.Second

func serveAndDrain(
	signalCtx context.Context,
	server *http.Server,
	listener net.Listener,
	closeResources func() error,
) error {
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	var serveErr error
	serveJoined := false
	select {
	case <-signalCtx.Done():
	case serveErr = <-serveDone:
		serveJoined = true
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	shutdownErr := server.Shutdown(drainCtx)
	cancel()

	var closeErr error
	if shutdownErr != nil {
		closeErr = server.Close()
	}
	if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		closeErr = errors.Join(closeErr, err)
	}
	if !serveJoined {
		serveErr = <-serveDone
	}
	if errors.Is(serveErr, http.ErrServerClosed) || errors.Is(serveErr, net.ErrClosed) {
		serveErr = nil
	}
	if shutdownErr != nil {
		return errors.Join(shutdownErr, closeErr, serveErr)
	}
	return errors.Join(serveErr, closeErr, closeResources())
}
~~~

В существующей точке входа корневой сигнал можно связать с контекстом так; на Unix-сервисе добавьте нужные целевые сигналы, например SIGTERM:

~~~go
signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()
return serveAndDrain(signalCtx, server, listener, closeResources)
~~~

В loopback-проверке через `httptest` активный handler удерживался до снятия барьера: до его возврата ресурс оставался открыт; затем `Shutdown` завершил drain, `Serve` вернул штатный `http.ErrServerClosed`, а функция закрытия ресурса была вызвана после join. Так проверяются порядок и результат без обращения к внешней сети.

## Ошибки и ограничения

- Не передавайте уже отменённый signal context в `Server.Shutdown`. Он должен управлять переходом к остановке, а не сокращать бюджет drain до нуля.
- Не вызывайте `os.Exit` сразу при получении сигнала: отложенная очистка не выполняется, а gorутины не присоединяются.
- Не закрывайте базу или общий клиент до того, как завершены обработчики и фоновые workers, которые от них зависят.
- Не считайте `Shutdown` гарантией закрытия WebSocket, hijacked-соединений или своих фоновых циклов.
- Не проглатывайте превышение deadline. Запишите причину, выполните согласованный force-close и оставьте внешний предел процесса конечным.
- Не отменяйте сразу рабочий контекст, если принятые задачи должны drain-иться; сначала остановите их admission и определите политику для очереди.

Диагностируйте задержку по этапам: время до остановки listener, число активных handlers, оставшиеся workers и последний ресурс в ожидании. Метрика «процесс жив» не показывает, какой этап держит shutdown. Для HTTP-деталей и локальных тестов см. [«HTTP»](../stdlib/net-http.md), а для порядка отмены и join — [«Жизненный цикл горутины»](../concurrency/goroutine-lifecycle.md) и [«Context и отмена»](../concurrency/context.md).

## Самопроверка

<details>
<summary>Почему контекст сигнала не подходит для ожидания активных обработчиков?</summary>

После получения сигнала он уже отменён, поэтому его deadline/cancellation немедленно оборвёт drain. Нужен новый контекст с конечным timeout и независимым родителем. Senior follow-up: откуда взять значения трассировки, если shutdown не должен наследовать отмену запроса?
</details>

<details>
<summary>Что остаётся присоединить после успешного Server.Shutdown?</summary>

Главную горутину, которая выполняла Serve, и принадлежащие приложению фоновые задачи; затем можно закрыть общие зависимости. Senior follow-up: что именно возвращает Serve в штатном случае и где нужно прочитать это значение?
</details>

<details>
<summary>Можно ли закрыть базу сразу после Server.Close при истёкшем drain timeout?</summary>

Не автоматически: Close закрывает соединения, но handler-код мог продолжить выполняться и пользоваться базой. Сначала используйте отдельный join приложения; если он не укладывается в бюджет, зафиксируйте failure и примените согласованную политику процесса. Senior follow-up: какой общий shutdown deadline и какой финальный предел задаёт ваш supervisor?
</details>

## Практика

Возьмите handler с медленным локальным запросом через `httptest` и проверьте, что вызов `Shutdown` закрывает приём новых соединений, но ждёт handler в пределах timeout. Затем проверьте таймаут и force-close отдельно, не закрывая тестовую общую зависимость до подтверждённого join. Добавьте тест, в котором фоновый worker дренирует очередь после остановки HTTP admission. Сравните HTTP-контракт статьи [«HTTP»](../stdlib/net-http.md) и управление задачами в [«Жизненном цикле горутины»](../concurrency/goroutine-lifecycle.md).

## Источники

- [http.Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown) — описывает закрытие listener-ов, ожидание handlers, результат Serve и исключение для hijacked-соединений.
- [signal.NotifyContext](https://pkg.go.dev/os/signal#NotifyContext) — задаёт связь системного сигнала с отменяемым контекстом и роль stop-функции.
- [context.WithTimeout](https://pkg.go.dev/context#WithTimeout) — создаёт конечный drain deadline; в примере родителем служит новый Background-контекст.
- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines) — объясняет, почему остановившийся потребитель должен сообщить upstream-этапам об отмене и затем дождаться их завершения.
