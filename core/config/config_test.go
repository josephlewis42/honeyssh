package config

import (
	"bytes"
	"compress/gzip"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestBuiltinConfig(t *testing.T) {
	rawConfig := make(map[string]interface{})
	assert.Nil(t, yaml.Unmarshal(defaultConfigData, &rawConfig))

	knownFields := make(map[string]bool)
	rt := reflect.TypeOf(Configuration{})
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		assert.NotEmpty(t, jsonTag)
		jsonField := strings.Split(jsonTag, ",")[0]
		knownFields[jsonField] = true

		if _, ok := rawConfig[jsonField]; !ok {
			assert.False(t, true, "default config missing field: %q", jsonField)
		}
	}

	for k := range rawConfig {
		_, ok := knownFields[k]
		assert.True(t, ok, "default config contains invalid field: %q", k)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Will panic() on load failure because it should never happen at runtime.
	assert.NotNil(t, defaultConfig())
}

func TestDefaultPasswords(t *testing.T) {
	// Will panic() on load failure because it should never happen at runtime.
	assert.NotNil(t, defaultPasswords())
}

func TestFs(t *testing.T) {
	fsReader := bytes.NewReader(rootFsData)
	_, gzipErr := gzip.NewReader(fsReader)
	assert.Nil(t, gzipErr, "not a valid gzip")
}
