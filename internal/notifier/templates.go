package notifier

import "fmt"

func SupportMessage(support Support) string {
	if !support.Enabled || len(support.Links) == 0 {
		return ""
	}
	message := `
<tg-spoiler>
Если сервис оказался полезным, вы можете поддержать его развитие:
`
	for _, link := range support.Links {
		message += fmt.Sprintf("\n🔗 <a href=\"%s\">%s</a>", link.URL, link.Title)
	}
	message += `
Спасибо за поддержку ❤️
</tg-spoiler>`
	return message
}

func FailoverMessage(fromCountry, toCountry string, support Support) string {
	fromFlag, fromName := countryInfo(fromCountry)
	toFlag, toName := countryInfo(toCountry)
	return fmt.Sprintf(
		`%s WARP %s

На сервере возникли неполадки.
Чтобы вы могли продолжать пользоваться сервисов без перебоев,
мы временно перенаправили трафик на %s %s.
Соединение продолжает работать в штатном режиме.

Все работает штатно. Как только основной сервер восстановится,
мы автоматически вернем маршрут обратно,
%s`,
		fromFlag, fromName,
		toFlag, toName,
		SupportMessage(support),
	)
}

func RecoveryMessage(toCountry string, support Support) string {
	toFlag, toName := countryInfo(toCountry)
	return fmt.Sprintf(
		`%s WARP %s

Основной сервер восстановлен.

Мы вернули трафик обратно на основной маршрут.

Все в порядке, можете продолжать пользоваться сервисом.
%s`,
		toFlag, toName,
		SupportMessage(support),
	)
}