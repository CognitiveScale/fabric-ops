package deploy

import "github.com/spf13/viper"

type Manifest struct {
	Agents  []string
	Skills  []string
	Actions []string
}

func NewManifest(configPath string) Manifest {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.ReadInConfig()
	var manifest Manifest
	viper.UnmarshalKey("cortex", &manifest)
	return manifest
}
