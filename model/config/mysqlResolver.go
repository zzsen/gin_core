package config

type DbResolver struct {
	Sources  []DbInfo `yaml:"sources"`
	Replicas []DbInfo `yaml:"replicas"`
	Tables   []string `yaml:"tables"`
}
