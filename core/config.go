package core

import "path/filepath"

// https://cloudinit.readthedocs.io/en/latest/topics/instancedata.html
type Configuration struct {
	Motd          string `json:"motd"`
	SSHPort       int    `json:"ssh_port"`
	Hostname      string `json:"hostname"`
	RootFsTarPath string `json:"root_fs_tar_path"`
	HostKeyPath   string `json:"host_key_path"`
	LogPath       string `json:"log_path"`
}

func DefaultConfig() *Configuration {
	return &Configuration{
		Motd:          `Last login: Sun Jun 27 16:19:57 PDT 2021 on tty1`,
		SSHPort:       2222,
		Hostname:      "localhost",
		RootFsTarPath: "",
		HostKeyPath:   "", // Genreate a key.
		LogPath:       filepath.Join(".", "logs"),
	}
}
