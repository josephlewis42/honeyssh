package vos

type VNetwork interface {
	Hostname() (string, error)
}

// VOS provides a virtual OS interface.
type VOS interface {
	VNetwork
	VEnv
	VIO
	VProc
	VFS

	StartProcess(name string, argv []string, attr *ProcAttr) (VOS, error)
}
