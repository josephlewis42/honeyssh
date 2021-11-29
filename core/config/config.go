package config

import "path/filepath"

const ConfigurationName = "config.yaml"

// https://cloudinit.readthedocs.io/en/latest/topics/instancedata.html
type Configuration struct {
	configurationDir string

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

func defaultConfig() *Configuration {
	return &Configuration{
		Motd:     `Last login: Sun Jun 27 16:19:57 PDT 2021 on tty1`,
		SSHPort:  2222,
		Hostname: "localhost",
	}
}
