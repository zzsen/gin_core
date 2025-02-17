package config

type EsInfo struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
}
