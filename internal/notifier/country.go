package notifier

func GetCountry(code string) Country {

	country, ok := Countries[code]

	if !ok {

		return Country{
			Name: code,
			Flag: "🏳️",
		}
	}

	return country
}
