package deploy

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type Manifest struct {
	Cortex struct {
		Agent     []string
		Skill     []string
		Action    []string
		Snapshots []string
		Type      []string

		Experiment []string
		Model      []string
		Run        []string

		Connection []string
		Campaign   []string

		Dependencies map[string]interface{} `yaml:"_dependencies"`
	} `yaml: "cortex"`
}

func NewManifest(configPath string) Manifest {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln("Failed to read manifest file ", configPath, " Error: ", err)
	}

	var manifest Manifest
	err = yaml.Unmarshal(yamlFile, &manifest)
	if err != nil {
		log.Fatalln("Failed to parse manifest file ", configPath, " Error: ", err)
	}
	return manifest
}
