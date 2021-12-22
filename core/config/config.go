package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
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
	configFs afero.Fs

	Motd             string `json:"motd"`
	SSHPort          int    `json:"ssh_port" validate:"gte=0,lte=65535"`
	SSHBanner        string `json:"ssh_banner"`
	AllowAnyPassword bool   `json:"allow_any_password"`

	GlobalPasswords []string `json:"global_passwords"`

	OS OS `json:"os"`

	Users []User `json:"users" validate:"unique=Username"`

	Uname Uname `json:"uname"`
}

// Validate the configuration for basic semantic errors.
func (c *Configuration) Validate() error {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		return name
	})

	return validate.Struct(c)
}

type User struct {
	Username  string   `json:"username" validate:"required"`
	UID       int      `json:"uid" validate:"gte=0"`
	GID       int      `json:"gid" validate:"gte=0"`
	Home      string   `json:"home" validate:"required"`
	Shell     string   `json:"shell" validate:"required"`
	Passwords []string `json:"passwords" validate:"unique"`
}

type OS struct {
	DefaultShell string `json:"default_shell" validate:"required"`
	DefaultPath  string `json:"default_path" validate:"required"`
}

type Uname struct {
	KernelName       string `json:"kernel_name" validate:"required"`               // Kernel Name name e.g. "Linux".
	Nodename         string `json:"nodename" validate:"required,hostname_rfc1123"` // Hostname of the machine on one of its networks.
	KernelRelease    string `json:"kernel_release" validate:"required"`            // OS release e.g. "4.15.0-147-generic"
	KernelVersion    string `json:"kernel_version" validate:"required"`            // OS version e.g. "#151-Ubuntu SMP Fri Jun 18 19:21:19 UTC 2021"
	HardwarePlatform string `json:"hardware_platform" validate:"required"`         // Machnine name e.g. "x86_64"
	Domainname       string `json:"domainname" validate:""`                        // NIS or YP domain name.
}

func (c *Configuration) fs() afero.Fs {
	return c.configFs
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

func (c *Configuration) ReadAppLog() (afero.File, error) {
	return c.fs().OpenFile(AppLogName, os.O_RDONLY, 0600)
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
