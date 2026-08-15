package model

type Prometheus struct {
	Job      string `yaml:"job"`
	Instance string `yaml:"instance"`
}

type DNS struct {
	Zone   string `yaml:"zone"`
	Record string `yaml:"record"` // полное доменное имя записи
	IP     string `yaml:"ip"`     // основной IP-адрес
}

type Host struct {
	Name       string     `yaml:"name"`
	Country    string     `yaml:"country"`
	Prometheus Prometheus `yaml:"prometheus"`
	DNS        DNS        `yaml:"dns"`
}