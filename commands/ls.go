package commands

import (
	"archive/tar"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	fcolor "github.com/fatih/color"
	"github.com/josephlewis42/honeyssh/core/vos"
	"github.com/josephlewis42/honeyssh/third_party/memmapfs/mem"
	getopt "github.com/pborman/getopt/v2"
)

// Ls implements the UNIX ls command.
func Ls(virtOS vos.VOS) int {

	// TODO: look up the actual GID
	gid2name := func(gid int) string {
		switch gid {
		case 0:
			return "root"
		default:
			return fmt.Sprintf("%d", gid)
		}
	}

	opts := getopt.New()
	listAll := opts.Bool('a', "don't ignore entries starting with .")
	longListing := opts.Bool('l', "use a long listing format")
	humanSize := opts.BoolLong("human-readable", 'h', "print human readable sizes")
	lineWidth := opts.IntLong("width", 'w', virtOS.GetPTY().Width, "set the column width, 0 is infinite")
	helpOpt := opts.BoolLong("help", '?', "show help and exit")

	var color ColorPrinter
	color.Init(opts, virtOS)

	if err := opts.Getopt(virtOS.Args(), nil); err != nil || *helpOpt {
		w := virtOS.Stderr()
		if err != nil {
			virtOS.LogInvalidInvocation(err)
			fmt.Fprintln(w, err)
		}
		fmt.Fprintln(w, "Usage: ls [OPTION]... [FILE]...")
		fmt.Fprintln(w, "List information about the FILEs (the current directory by default).")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		opts.PrintOptions(w)
		return 1
	}

	// Initialize arguments
	directoriesToList := opts.Args()
	if len(directoriesToList) == 0 {
		directoriesToList = append(directoriesToList, ".")
	}
	sort.Strings(directoriesToList)

	showDirectoryNames := len(directoriesToList) > 1

	sizeFmt := func(bytes int64) string {
		return fmt.Sprintf("%d", bytes)
	}
	if *humanSize {
		sizeFmt = BytesToHuman
	}

	if *lineWidth == 0 {
		*lineWidth = math.MaxInt32
	}

	uid2name := UidResolver(virtOS)

	exitCode := 0

	for _, directory := range directoriesToList {

		file, err := virtOS.Open(directory)
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "%s: %v\n", directory, err)
			exitCode = 1
			continue
		}

		allPaths, err := file.Readdir(-1)
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "%s: %v\n", directory, err)
			exitCode = 1
			continue
		}

		// TODO: add . and .. if -a is specified

		var totalSize int64
		var paths []os.FileInfo
		var longestNameLength int
		for _, path := range allPaths {
			if *listAll == false && strings.HasPrefix(path.Name(), ".") {
				continue
			}
			paths = append(paths, path)
			totalSize += path.Size()
			if l := len(path.Name()); l > longestNameLength {
				longestNameLength = l
			}
		}

		sort.Slice(paths, func(i int, j int) bool {
			return paths[i].Name() < paths[j].Name()
		})

		if showDirectoryNames {
			fmt.Fprintf(virtOS.Stdout(), "%s:\n", directory)
		}

		if *longListing {
			fmt.Fprintf(virtOS.Stdout(), "total %d\n", totalSize)
			tw := tabwriter.NewWriter(virtOS.Stdout(), 0, 0, 1, ' ', 0)
			for _, f := range paths {
				// TODO: number of hard links is better approximated by
				// 2 (self + parent) for a directory plus number of direct child
				// directories.
				hardLinks := 1
				if f.IsDir() {
					hardLinks = 2
				}

				// Include time if current year.
				currentYear := time.Now().Year()
				modTime := f.ModTime().Format("Jan _2 2006")
				if f.ModTime().Year() >= currentYear {
					modTime = f.ModTime().Format("Jan _2 15:04")
				}

				uid, gid := getUIDGID(f)
				fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
					f.Mode().String(),
					hardLinks,
					uid2name(uid),
					gid2name(gid),
					sizeFmt(f.Size()),
					modTime,
					color.Sprintf(Dircolor(f), f.Name()))
			}
			tw.Flush()
		} else {
			const minPaddingWidth = 2
			colWidths := columnize(paths, *lineWidth)
			cols := len(colWidths)
			rows := len(paths) / cols
			if len(paths)%cols > 0 {
				rows++
			}

			tw := virtOS.Stdout()
			for row := 0; row < rows; row++ {
				for col, width := range colWidths {
					// Add padding if there was a column befor this.
					if col > 0 {
						fmt.Fprintf(tw, "  ")
					}
					// Find and print the file entry.
					if index := (col * rows) + row; index < len(paths) {
						entry := paths[index]
						name := entry.Name()
						width -= len(name) // Subtract off padding.
						fmt.Fprintf(tw, color.Sprintf(Dircolor(entry), name))
					}
					// Add padding for alignment.
					if width > 0 {
						fmt.Fprintf(tw, strings.Repeat(" ", width))
					}
				}
				fmt.Fprintln(tw)
			}
		}
	}

	return exitCode
}

type LsColorTest struct {
	color *fcolor.Color
	test  func(fileInfo os.FileInfo) bool
}

// Color listing comes from: https://askubuntu.com/a/884513
var dircolors = []LsColorTest{
	// Directories are bold blue.
	{color: ColorBoldBlue, test: os.FileInfo.IsDir},
	// Symlinks are bold cyan.
	{color: ColorBoldCyan, test: func(fi os.FileInfo) bool {
		return fi.Mode()&fs.ModeSymlink > 0
	}},
	// Yellow with black background pipe, block device, char device.
	{color: fcolor.New(fcolor.FgYellow, fcolor.BgBlack, fcolor.Bold), test: func(fi os.FileInfo) bool {
		return fi.Mode()&(fs.ModeSymlink|fs.ModeDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeCharDevice) > 0
	}},
	// Executables are bold green.
	{color: ColorBoldGreen, test: func(fi os.FileInfo) bool {
		return fi.Mode().Perm()&0111 > 0
	}},
	// Archives are bold red.
	{color: ColorBoldRed, test: func(fi os.FileInfo) bool {
		return map[string]bool{
			"tar": true,
			"tgz": true,
			"zip": true,
			"gz":  true,
			"bz2": true,
			"bz":  true,
			"tbz": true,
			"deb": true,
			"rpm": true,
			"jar": true,
			"war": true,
			"rar": true,
		}[path.Ext(fi.Name())]
	}},
}

func Dircolor(fileInfo os.FileInfo) *fcolor.Color {
	for _, dc := range dircolors {
		if dc.test(fileInfo) {
			return dc.color
		}
	}

	// Anything else defaults to white.
	return fcolor.New(fcolor.FgHiWhite)
}

func columnize(paths []fs.FileInfo, screenWidth int) []int {
	numFiles := len(paths)
	if numFiles == 0 {
		return []int{0}
	}

	const colPadding = 2

	// Size of the display of the file name, actual length may vary if there are
	// escape sequences to format it.
	displayLengths := make([]int, len(paths))
	for i, p := range paths {
		displayLengths[i] = len(p.Name())
	}

	// Start with maximum number of columns and work down until all the data fits.
	// 3 is the minimum column width, 1 char filename + 2 padding.
	columns := screenWidth / (1 + colPadding)
	if columns > len(paths) {
		columns = len(paths)
	}
	var maximums []int // Holds maximum size of a name in the column.
	for ; columns >= 1; columns-- {
		maximums = make([]int, columns)
		total := (columns - 1) * colPadding
		rows := (numFiles / columns) + 1
		for i, nameLen := range displayLengths {
			prevMax := maximums[i/rows]
			if nameLen > prevMax {
				maximums[i/rows] = nameLen
				total = total - prevMax + nameLen
				if total > screenWidth {
					break
				}
			}
		}

		if total <= screenWidth {
			return maximums
		}
	}

	return maximums
}

func getUIDGID(fileInfo os.FileInfo) (uid, gid int) {
	switch v := (fileInfo.Sys()).(type) {
	case *syscall.Stat_t:
		return int(v.Uid), int(v.Gid)

	case tar.Header:
		return v.Uid, v.Gid

	case *tar.Header:
		return v.Uid, v.Gid

	case mem.FileInfo:
		return 1000, 1000

	case *mem.FileInfo:
		return 2000, 2000
	default:
		// TODO: Log the type that caused the failure.
		return 0, 0
	}
}

var _ vos.ProcessFunc = Ls

func init() {
	mustAddBinCmd("ls", Ls)
}
