package config

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	tempDir := t.TempDir()
	if err := Initialize(tempDir, log.New(ioutil.Discard, "", 0)); err != nil {
		t.Fatal(err)
	}

	// Check that the config is valid
	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateDownload", func(t *testing.T) {
		fd, err := cfg.CreateDownload("test")
		assert.Nil(t, err)
		fd.Close()
	})

	t.Run("CreateSessionLog", func(t *testing.T) {
		fd, err := cfg.CreateSessionLog("attacker.log")
		assert.Nil(t, err)
		fd.Close()
	})

	t.Run("OpenAppLog", func(t *testing.T) {
		fd, err := cfg.OpenAppLog()
		assert.Nil(t, err)
		fd.Close()
	})

	t.Run("OpenFilesystemTarGz", func(t *testing.T) {
		fd, err := cfg.OpenFilesystemTarGz()
		assert.Nil(t, err)
		fd.Close()
	})

	t.Run("PrivateKeyPem", func(t *testing.T) {
		keyPem, err := cfg.PrivateKeyPem()
		assert.Nil(t, err)
		assert.NotNil(t, keyPem)
	})
}
