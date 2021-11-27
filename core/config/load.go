package config

import (
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// Load loads the configuration from the directory.
func Load(path string) (*Configuration, error) {
	// If given the path to a config.yaml file, move back up a level.
	if filepath.Base(path) == ConfigurationName {
		path = filepath.Dir(path)
	}

	configContents, err := ioutil.ReadFile(filepath.Join(path, ConfigurationName))
	if err != nil {
		return nil, err
	}
	var out Configuration
	if err := yaml.UnmarshalStrict(configContents, &out); err != nil {
		return nil, err
	}
	out.configurationDir = path
	return &out, nil
}
