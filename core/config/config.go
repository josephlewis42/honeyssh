package config

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

var (
	//go:embed default/config.yaml
	defaultConfigData []byte

	//go:embed default/root_fs.tar.gz
	rootFsData []byte
)

const (
	ConfigurationName = "config.yaml"
	DownloadDirName   = "downloads"
	LogsDirName       = "session_logs"
	PrivateKeyName    = "private_key"
	RootFSName        = "root_fs.tar.gz"
	AppLogName        = "app.log"
)

type Configuration struct {
	configurationDir string
	configFs         afero.Fs

	Motd             string `json:"motd"`
	SSHPort          int    `json:"ssh_port"`
	SSHBanner        string `json:"ssh_banner"`
	AllowAnyPassword bool   `json:"allow_any_password"`

	GlobalPasswords []string `json:"global_passwords"`

	Users []User `json:"users"`

	Uname Uname `json:"uname"`
}

type User struct {
	Username  string   `json:"username"`
	UID       int      `json:"uid"`
	GID       int      `json:"gid"`
	Home      string   `json:"home"`
	Shell     string   `json:"shell"`
	Passwords []string `json:"passwords"`
}

type Uname struct {
	KernelName       string `json:"kernel_name"`       // Kernel Name name e.g. "Linux".
	Nodename         string `json:"nodename"`          // Hostname of the machine on one of its networks.
	KernelRelease    string `json:"kernel_release"`    // OS release e.g. "4.15.0-147-generic"
	KernelVersion    string `json:"kernel_version"`    // OS version e.g. "#151-Ubuntu SMP Fri Jun 18 19:21:19 UTC 2021"
	HardwarePlatform string `json:"hardware_platform"` // Machnine name e.g. "x86_64"
	Domainname       string `json:"domainname"`        // NIS or YP domain name.
}

func (c *Configuration) fs() afero.Fs {
	if c.configFs != nil {
		return c.configFs
	}

	return afero.NewBasePathFs(afero.NewOsFs(), c.configurationDir)
}

// Create a download with the given name.
func (c *Configuration) CreateDownload(name string) (afero.File, error) {
	toCreate := filepath.Join(DownloadDirName, name)
	return c.fs().Create(toCreate)
}

func (c *Configuration) CreateSessionLog(name string) (afero.File, error) {
	toCreate := filepath.Join(LogsDirName, name)
	return c.fs().Create(toCreate)
}

// PrivateKeyPem returns the bytes of the private key.
func (c *Configuration) PrivateKeyPem() ([]byte, error) {
	return afero.ReadFile(c.fs(), PrivateKeyName)
}

// OpenAppLog opens the application log in an append only state.
func (c *Configuration) OpenAppLog() (afero.File, error) {
	return c.fs().OpenFile(AppLogName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}

// OpenFilesystemTarGz opens the backing filesystem .tar.gz file.
func (c *Configuration) OpenFilesystemTarGz() (afero.File, error) {
	return c.fs().Open(RootFSName)
}

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
