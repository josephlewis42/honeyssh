package config

import (
	_ "embed"
	"path/filepath"
	"sync"

	"sigs.k8s.io/yaml"
)

var (
	//go:embed default/config.yaml
	defaultConfigData []byte

	//go:embed default/root_fs.tar.gz
	rootFsData []byte
)

const ConfigurationName = "config.yaml"

type Configuration struct {
	configurationDir string
	passwordLock     sync.Mutex
	cachedPasswords  map[string][]string

	Motd             string `json:"motd"`
	SSHPort          int    `json:"ssh_port"`
	Hostname         string `json:"hostname"`
	SSHBanner        string `json:"ssh_banner"`
	AllowAnyPassword bool   `json:"allow_any_password"`

	GlobalPasswords []string `json:"global_passwords"`

	Users []User `json:"users"`
}

type User struct {
	Username  string   `json:"username"`
	UID       int      `json:"uid"`
	GID       int      `json:"gid"`
	Home      string   `json:"home"`
	Shell     string   `json:"shell"`
	Passwords []string `json:"passwords"`
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

// HostKeyPath holds the path to the host keys.
func (c *Configuration) HostKeyPath() string {
	return filepath.Join(c.configurationDir, "private_key")
}

// RootFsTarPath holds the path to the root FS.
func (c *Configuration) RootFsTarPath() string {
	return filepath.Join(c.configurationDir, "root_fs.tar.gz")
}

type passwordsData map[string][]string

// GetPasswords returns allowable passwords for the given username.
func (c *Configuration) GetPasswords(username string) []string {
	var out []string
	for _, v := range c.Users {
		if v.Username == username {
			out = append(out, v.Passwords...)
		}
	}

	out = append(out, c.GlobalPasswords...)
	return out
}

func defaultConfig() *Configuration {
	var out Configuration
	if err := yaml.UnmarshalStrict(defaultConfigData, &out); err != nil {
		panic(err)
	}
	return &out
}
