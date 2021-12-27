package vos

import (
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"josephlewis.net/honeyssh/third_party/memmapfs"
)

func FSTestCase(t *testing.T, suite FSTestSuite, testPath string) *FSTestCaseSetup {
	testFS, checkFS := suite.MakeFS(t)

	prefixer := func(in string) string {
		return in
	}
	if suite.Prefixer != nil {
		prefixer = suite.Prefixer
	}

	return &FSTestCaseSetup{
		check: &FSTestCaseCheck{
			t:    t,
			fs:   checkFS,
			name: testPath,
		},

		t:        t,
		fs:       testFS,
		testPath: testPath,
		prefixer: prefixer,
	}
}

func (tc *FSTestCaseSetup) MkdirTestPath(perm fs.FileMode) *FSTestCaseSetup {
	return tc.Mkdir(tc.testPath, perm)
}

func (tc *FSTestCaseSetup) Mkdir(path string, perm fs.FileMode) *FSTestCaseSetup {
	if err := tc.fs.Mkdir(tc.prefixer(path), perm); err != nil {
		tc.t.Fatal(err)
	}

	return tc
}

func (tc *FSTestCaseSetup) MkdirAllParentsTestPath(perm fs.FileMode) *FSTestCaseSetup {
	return tc.MkdirAllParents(tc.testPath, perm)
}

func (tc *FSTestCaseSetup) MkdirAllParents(name string, perm fs.FileMode) *FSTestCaseSetup {
	if err := tc.fs.MkdirAll(tc.prefixer(path.Dir(name)), perm); err != nil {
		tc.t.Fatal(err)
	}

	return tc
}

func (tc *FSTestCaseSetup) CreateTestPath() *FSTestCaseSetup {
	return tc.Create(tc.testPath)
}

func (tc *FSTestCaseSetup) Create(path string) *FSTestCaseSetup {
	fd, err := tc.fs.Create(tc.prefixer(path))
	if err != nil {
		tc.t.Fatal(err)
	}
	fd.Close()

	return tc
}

func (tc *FSTestCaseSetup) AssertAfter(callback func(fs VFS, name string) error) *FSTestCaseCheck {
	tc.check.err = callback(tc.fs, tc.prefixer(tc.testPath))
	return tc.check
}

type FSTestCaseSetup struct {
	check *FSTestCaseCheck

	t        *testing.T
	fs       VFS
	testPath string
	prefixer func(string) string
}

type FSTestCaseCheck struct {
	t    *testing.T
	fs   VFS
	name string
	err  error
}

func (tc *FSTestCaseCheck) NoError() *FSTestCaseCheck {
	assert.Nil(tc.t, tc.err)
	return tc
}

func (tc *FSTestCaseCheck) Error() *FSTestCaseCheck {
	assert.Error(tc.t, tc.err)
	return tc
}

func (tc *FSTestCaseCheck) ErrorIs(desired error) *FSTestCaseCheck {
	assert.ErrorIs(tc.t, tc.err, desired)
	return tc
}

func (tc *FSTestCaseCheck) OutExists() *FSTestCaseCheck {
	return tc.Exists(tc.name)
}

func (tc *FSTestCaseCheck) Exists(name string) *FSTestCaseCheck {
	exists, err := afero.Exists(tc.fs, name)
	if err != nil {
		tc.t.Errorf("exists %q: %v", name, err)
	}
	if !exists {
		tc.t.Errorf("doesn't exist: %q", name)
	}

	return tc
}

func (tc *FSTestCaseCheck) TestPathIsDir() *FSTestCaseCheck {
	return tc.IsDir(tc.name)
}

func (tc *FSTestCaseCheck) IsDir(name string) *FSTestCaseCheck {
	info, err := tc.fs.Stat(name)
	if err != nil {
		tc.t.Errorf("stat %q: %v", name, err)
	}
	assert.True(tc.t, info.IsDir(), "IsDir()")

	return tc
}

type FSTestSuite struct {
	// MakeFS creates an FS for a single test. In is the FS that will be operated
	// on with the test. out is the FS checked for data. If error is set, the test
	// will fail.
	MakeFS func(t *testing.T) (in, out VFS)

	// Prefixer adds a prefix to a test entry. Input paths will ALWAYS be absolute
	// and slash delimited.
	Prefixer func(name string) (outname string)

	// Whether the FS supports symlinks.
	SupportsSymlinks bool
}

func RunFsTest(t *testing.T, suite FSTestSuite) {
	t.Run("Create", func(t *testing.T) {
		callback := func(fs VFS, name string) error {
			_, err := fs.Create(name)
			return err
		}

		t.Run("nominal", func(t *testing.T) {
			FSTestCase(t, suite, "/note.txt").
				AssertAfter(callback).
				NoError().
				OutExists()
		})
		t.Run("exists", func(t *testing.T) {
			// Create should work over existing files.
			FSTestCase(t, suite, "/note.txt").
				CreateTestPath().
				AssertAfter(callback).
				NoError().
				OutExists()
		})
		t.Run("exists as a dir", func(t *testing.T) {
			// Create should fail over directories.
			FSTestCase(t, suite, "/note").
				MkdirTestPath(0700).
				AssertAfter(callback).
				Error()
		})
		t.Run("missing dir", func(t *testing.T) {
			FSTestCase(t, suite, "/does/not/exist/note").
				AssertAfter(callback).
				ErrorIs(fs.ErrNotExist)
		})
		t.Run("nested", func(t *testing.T) {
			FSTestCase(t, suite, "/path/that/exists/note").
				MkdirAllParentsTestPath(0700).
				AssertAfter(callback).
				NoError().
				OutExists()
		})

		// TODO: create under a link
	})

	t.Run("Mkdir", func(t *testing.T) {
		mkdirCallback := func(fs VFS, name string) error {
			return fs.Mkdir(name, 0700)
		}

		t.Run("nominal", func(t *testing.T) {
			FSTestCase(t, suite, "/dir").
				AssertAfter(mkdirCallback).
				NoError().
				TestPathIsDir()
		})
		t.Run("exists", func(t *testing.T) {
			// Create should work over existing files.
			FSTestCase(t, suite, "/dir").
				MkdirTestPath(0777).
				AssertAfter(mkdirCallback).
				ErrorIs(fs.ErrExist).
				TestPathIsDir()
		})
		t.Run("exists as file", func(t *testing.T) {
			// Create should fail over directories.
			FSTestCase(t, suite, "/dir").
				CreateTestPath().
				AssertAfter(mkdirCallback).
				Error()
		})
		t.Run("missing dir", func(t *testing.T) {
			FSTestCase(t, suite, "/does/not/exist/dir").
				AssertAfter(mkdirCallback).
				ErrorIs(fs.ErrNotExist)
		})
		t.Run("nested", func(t *testing.T) {
			FSTestCase(t, suite, "/path/that/exists/note").
				MkdirAllParentsTestPath(0700).
				AssertAfter(mkdirCallback).
				NoError().
				OutExists()
		})

		// TODO: create under a link
	})
}

func TestLinkingFs(t *testing.T) {
	suite := FSTestSuite{
		MakeFS: func(t *testing.T) (VFS, VFS) {
			fs := NewLinkingFs(memmapfs.NewMemMapFs(time.Now))
			return fs, fs
		},
	}

	RunFsTest(t, suite)
}

func TestNewMemCopyOnWriteFs(t *testing.T) {
	t.Skip("afero's union FS is broken")
	suite := FSTestSuite{
		MakeFS: func(t *testing.T) (VFS, VFS) {
			mfs := NewLinkingFs(memmapfs.NewMemMapFs(time.Now))
			fs := NewMemCopyOnWriteFs(mfs, time.Now)
			return fs, fs
		},
	}

	RunFsTest(t, suite)

}

func TestMountFS(t *testing.T) {
	suite := FSTestSuite{
		MakeFS: func(t *testing.T) (VFS, VFS) {
			mfs := NewLinkingFs(memmapfs.NewMemMapFs(time.Now))
			fs := NewMountFS(mfs)
			return fs, fs
		},
	}

	RunFsTest(t, suite)
}

func TestOSFs(t *testing.T) {
	suite := FSTestSuite{
		MakeFS: func(t *testing.T) (VFS, VFS) {
			td := t.TempDir()
			t.Cleanup(func() {
				os.RemoveAll(td)
			})

			fs := afero.NewBasePathFs(afero.NewOsFs(), td)
			return fs, fs
		},
	}

	RunFsTest(t, suite)
}
