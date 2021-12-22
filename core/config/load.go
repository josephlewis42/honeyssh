package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// Load loads the configuration from the directory.
func Load(path string) (*Configuration, error) {
	// If given the path to a config.yaml file, move back up a level.
	if filepath.Base(path) == ConfigurationName {
		path = filepath.Dir(path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	configContents, err := ioutil.ReadFile(filepath.Join(absPath, ConfigurationName))
	if err != nil {
		return nil, err
	}
	var out Configuration
	if err := yaml.UnmarshalStrict(configContents, &out); err != nil {
		return nil, err
	}
	out.configFs = afero.NewBasePathFs(afero.NewOsFs(), absPath)

	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}
