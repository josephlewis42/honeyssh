package vos

import (
	"io"
	"os"
)

func NewVIOAdapter(stdin io.ReadCloser, stdout, stderr io.WriteCloser) *VIOAdapter {
	return &VIOAdapter{
		IStdin:  stdin,
		IStdout: stdout,
		IStderr: stderr,
	}
}

func NewNullIO() VIO {
	return NewVIOAdapter(&ClosedReader{}, &NopWriteCloser{}, &NopWriteCloser{})
}

type VIOAdapter struct {
	IStdin  io.ReadCloser
	IStdout io.WriteCloser
	IStderr io.WriteCloser
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

// ClosedReader implemnets io.Reader and always throws ErrClosed on Read.
type ClosedReader struct{}

var _ io.ReadCloser = (*ClosedReader)(nil)

func (*ClosedReader) Read([]byte) (int, error) {
	return 0, os.ErrClosed
}

func (*ClosedReader) Close() error {
	return nil
}

type NopWriteCloser struct{}

var _ io.WriteCloser = (*NopWriteCloser)(nil)

func (*NopWriteCloser) Write(b []byte) (int, error) {
	return len(b), nil
}

func (*NopWriteCloser) Close() error {
	return nil
}
