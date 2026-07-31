package notifier

import "fmt"

func FailoverMessage(fromFlag, fromName, toFlag, toName string) string {
	return fmt.Sprintf(
`%s WARP %s

Из-за недоступности сервера

трафик временно переключен на %s %s.

Соединение продолжает работать в штатном режиме.

Мы автоматически вернем маршрут после восстановления сервера.`,
		fromFlag,
		fromName,
		toFlag,
		toName,
	)
}