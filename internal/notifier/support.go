package notifier

type SupportLink struct {
	Title string
	URL   string
}

type Support struct {
	Enabled bool
	Links   []SupportLink
}
