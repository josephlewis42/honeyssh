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

// Counter counts the number of matching and total bytes through a reader.
type Counter struct {
	test    func(byte) bool
	wrapped io.ReadCloser

	// Total number of bytes read.
	Total int
	// Total number of matched bytes.
	MatchedTotal int
}

// NewCounter creates a new counter over the wrapped stream applying test to
// each byte to update the TestTotal property.
func NewCounter(wrapped io.ReadCloser, test func(byte) bool) *Counter {
	return &Counter{
		test:    test,
		wrapped: wrapped,
	}
}

var _ io.ReadCloser = (*Counter)(nil)

// Read implements io.Reader.
func (c *Counter) Read(data []byte) (int, error) {
	cnt, err := c.wrapped.Read(data)
	c.Total += cnt

	for i := 0; i < cnt; i++ {
		if c.test(data[i]) {
			c.MatchedTotal++
		}
	}

	return cnt, err
}

// Close implemnts io.Closer.
func (c *Counter) Close() error {
	return c.wrapped.Close()
}
