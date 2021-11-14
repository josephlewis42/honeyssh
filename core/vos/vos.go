package vos

type VNetwork interface {
	Hostname() (string, error)
}

type PTY struct {
	Width  int
	Height int
	Term   string
	IsPTY  bool
}

// VOS provides a virtual OS interface.
type VOS interface {
	VNetwork
	VEnv
	VIO
	VProc
	VFS

	SetPTY(PTY)
	GetPTY() PTY
	StartProcess(name string, argv []string, attr *ProcAttr) (VOS, error)
}
