package model

// Record описывает DNS-запись (A или CNAME)
type Record struct {
	ID     string `json:"id"`      // ID записи в DNS-провайдере
	Name   string `json:"name"`    // полное доменное имя
	Type   string `json:"type"`    // "A" или "CNAME"
	IP     string `json:"ip"`      // для A-записи
	Target string `json:"target"`  // для CNAME
	TTL    int    `json:"ttl"`
}