package config

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DbResolver struct {
	Sources  []DbInfo `yaml:"sources"`
	Replicas []DbInfo `yaml:"replicas"`
	Tables   []any    `yaml:"tables"`
}

func (dbResolver *DbResolver) IsValid() bool {
	return len(dbResolver.Sources) > 0
}

func (dbResolver *DbResolver) SourceConfigs() []gorm.Dialector {
	sourceConfigs := []gorm.Dialector{}
	for _, source := range dbResolver.Sources {
		sourceConfigs = append(sourceConfigs, mysql.Open(source.Dsn()))
	}
	return sourceConfigs
}

func (dbResolver *DbResolver) ReplicaConfigs() []gorm.Dialector {
	replicaConfigs := []gorm.Dialector{}
	for _, replica := range dbResolver.Replicas {
		replicaConfigs = append(replicaConfigs, mysql.Open(replica.Dsn()))
	}
	return replicaConfigs
}

type DbResolvers []DbResolver

func (dbResolvers DbResolvers) IsValid() bool {
	for _, resolver := range dbResolvers {
		if !resolver.IsValid() {
			return false
		}
	}
	return true
}

func (dbResolvers DbResolvers) DefaultConfig() DbInfo {
	return dbResolvers[0].Sources[0]
}
