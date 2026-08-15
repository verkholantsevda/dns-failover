package notifier

type Country struct {
	Name string
	Flag string
}

var Countries = map[string]Country{

	"SE": {
		Name: "Sweden",
		Flag: "🇸🇪",
	},

	"FR": {
		Name: "France",
		Flag: "🇫🇷",
	},

	"DE": {
		Name: "Germany",
		Flag: "🇩🇪",
	},

	"EE": {
		Name: "Estonia",
		Flag: "🇪🇪",
	},
	"GB": {
		Name: "Greate Britian",
		Flag: "🇬🇧",
	},
}
