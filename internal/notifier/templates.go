package notifier

import "fmt"

func countryInfo(code string) (string, string) {

	country, ok := Countries[code]

	if !ok {
		return "🏳️", code
	}

	return country.Flag, country.Name
}

func SupportMessage(
	support Support,
) string {

	if !support.Enabled || len(support.Links) == 0 {
		return ""
	}

	message := `
<tg-spoiler>
💙 Поддержать проект:
Если сервис оказался полезным, вы можете поддержать его развитие:
`

	for _, link := range support.Links {

		message += fmt.Sprintf(
			"\n🔗 <a href=\"%s\">%s</a>",
			link.URL,
			link.Title,
		)
	}

	message += `
Спасибо за поддержку ❤️
</tg-spoiler>`

	return message
}

func FailoverMessage(
	fromCountry string,
	toCountry string,
	support Support,
) string {

	fromFlag, fromName := countryInfo(fromCountry)
	toFlag, toName := countryInfo(toCountry)

	return fmt.Sprintf(
	`%s WARP %s

Из-за недоступности сервера

трафик временно переключен на %s %s.
Соединение продолжает работать в штатном режиме.

Мы автоматически вернем маршрут после восстановления сервера.
%s`,
		fromFlag,
		fromName,
		toFlag,
		toName,
		SupportMessage(support),
	)
}

func RecoveryMessage(
	fromCountry string,
	toCountry string,
	support Support,
) string {

	fromFlag, fromName := countryInfo(fromCountry)
	toFlag, toName := countryInfo(toCountry)

	return fmt.Sprintf(
	`%s WARP %s

Основной сервер восстановлен.
Трафик возвращен обратно на %s %s.

Соединение работает в штатном режиме.

%s`,
		fromFlag,
		fromName,
		toFlag,
		toName,
		SupportMessage(support),
	)
}
