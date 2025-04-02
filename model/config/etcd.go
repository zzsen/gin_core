package config

type EtcdInfo struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Timeout   *int     `yaml:"timeout"`
}
