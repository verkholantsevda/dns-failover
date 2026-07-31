package notifier

var Countries = map[string]struct {
	Name string
	Flag string
}{
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
	"EST": {
		Name: "Estonia",
		Flag: "🇪🇪",
	},
}