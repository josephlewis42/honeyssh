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

func NewVIOAdapter(stdin io.Reader, stdout, stderr io.Writer) *VIOAdapter {
	return &VIOAdapter{
		IStdin:  toReadCloserOrDiscard(stdin),
		IStdout: toWriteCloserOrDiscard(stdout),
		IStderr: toWriteCloserOrDiscard(stderr),
	}
}

// NewNullIO creates a valid /dev/null style I/O, reads won't work and
// writes will be discarded.
func NewNullIO() VIO {
	return NewVIOAdapter(nil, nil, nil)
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

func toWriteCloserOrDiscard(w io.Writer) io.WriteCloser {
	if w == nil {
		return &devNull{}
	}
	if wc, ok := w.(io.WriteCloser); ok {
		return wc
	}

	return nopWriteCloser{w}
}

func toReadCloserOrDiscard(r io.Reader) io.ReadCloser {
	if r == nil {
		return &devNull{}
	}
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}

	return io.NopCloser(r)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

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
