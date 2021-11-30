package vos

import (
	"io"
	"os"
)

type VIOAdapter struct {
	IStdin  io.ReadCloser
	IStdout io.WriteCloser
	IStderr io.WriteCloser
}

func NewVIOAdapter(stdin io.ReadCloser, stdout, stderr io.WriteCloser) *VIOAdapter {
	return &VIOAdapter{
		IStdin:  stdin,
		IStdout: stdout,
		IStderr: stderr,
	}
}

// NewNullIO creates a valid /dev/null style I/O, reads won't work and
// writes will be discarded.
func NewNullIO() VIO {
	return NewVIOAdapter(&devNull{}, &devNull{}, &devNull{})
}

var _ VIO = (*VIOAdapter)(nil)

func (pr *VIOAdapter) Stdin() io.ReadCloser {
	return pr.IStdin
}

func (pr *VIOAdapter) Stdout() io.WriteCloser {
	return pr.IStdout
}

func (pr *VIOAdapter) Stderr() io.WriteCloser {
	return pr.IStderr
}

// devNull implemnets io.Reader and io.Writer, always closing for reads and
// discarding writes.
type devNull struct{}

var _ io.ReadCloser = (*devNull)(nil)
var _ io.WriteCloser = (*devNull)(nil)

func (*devNull) Read([]byte) (int, error) {
	return 0, os.ErrClosed
}

func (*devNull) Close() error {
	return nil
}

func (*devNull) Write(b []byte) (int, error) {
	return len(b), nil
}
