package model

type Record struct {
    ID       string
	TTL      int
	IP       string
	Country string
    Priority int
    Disabled bool
}