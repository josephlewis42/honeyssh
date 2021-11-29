package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"sigs.k8s.io/yaml"
)

const ConfigurationName = "config.yaml"

// https://cloudinit.readthedocs.io/en/latest/topics/instancedata.html
type Configuration struct {
	configurationDir string
	passwordLock     sync.Mutex
	cachedPasswords  map[string][]string

	Motd      string `json:"motd"`
	SSHPort   int    `json:"ssh_port"`
	Hostname  string `json:"hostname"`
	SSHBanner string `json:"ssh_banner"`
}

func (c *Configuration) ConfigurationPath() string {
	return filepath.Join(c.configurationDir, ConfigurationName)
}

// DownloadPath holds the path to the downloads relative to the configuraiton.
func (c *Configuration) DownloadPath() string {
	return filepath.Join(c.configurationDir, "downloads")
}

// LogPath holds the path to the CLI interaction logs.
func (c *Configuration) LogPath() string {
	return filepath.Join(c.configurationDir, "logs")
}

// AppLogPath holds the path to the application interaction logs.
func (c *Configuration) AppLogPath() string {
	return filepath.Join(c.configurationDir, "app.log")
}

// PasswordsPath holds the path to the list of passwords that will be accepted.
// Passwords associated with a "*" are allowed for all users.
func (c *Configuration) PasswordsPath() string {
	return filepath.Join(c.configurationDir, "passwords.yaml")
}

// HostKeyPath holds the path to the host keys.
func (c *Configuration) HostKeyPath() string {
	return filepath.Join(c.configurationDir, "private_key")
}

// RootFsTarPath holds the path to the root FS.
func (c *Configuration) RootFsTarPath() string {
	return filepath.Join(c.configurationDir, "root_fs.tar")
}

// GetPasswords returns allowable passwords for the given username.
func (c *Configuration) GetPasswords(username string) ([]string, error) {
	c.passwordLock.Lock()
	defer c.passwordLock.Unlock()

	if c.cachedPasswords == nil {
		passwordsRaw, err := ioutil.ReadFile(c.PasswordsPath())
		if err != nil {
			return nil, fmt.Errorf("no password file: %v", err)
		}
		c.cachedPasswords = make(map[string][]string)
		if err := yaml.UnmarshalStrict(passwordsRaw, &c.cachedPasswords); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal passwords file: %v", err)
		}
	}
	var out []string
	out = append(out, c.cachedPasswords[username]...)
	out = append(out, c.cachedPasswords["*"]...)
	return out, nil
}

func defaultConfig() *Configuration {
	return &Configuration{
		Motd:     `Last login: Sun Jun 27 16:19:57 PDT 2021 on tty1`,
		SSHPort:  2222,
		Hostname: "localhost",
	}
}
