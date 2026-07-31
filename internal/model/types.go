package model
type Prometheus struct {
	Job      string `yaml:"job"`
	Instance string `yaml:"instance"`
}


type DNS struct {
	Zone    string   `yaml:"zone"`
	Record  string   `yaml:"record"`
	Records []Record `yaml:"records"`
}

type Host struct {
	Name string `yaml:"name"`
	ICMPHost string `yaml:"icmp_host"`
	Prometheus Prometheus `yaml:"prometheus"`
	DNS DNS `yaml:"dns"`
}