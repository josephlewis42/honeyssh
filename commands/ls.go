package commands

import (
	"archive/tar"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"josephlewis.net/osshit/core/vos"
)

// Ls implements the UNIX ls command.
func Ls(virtOS vos.VOS) int {

	// TODO: look up the actual UID
	uid2name := func(uid int) string {
		switch uid {
		case 0:
			return "root"
		default:
			return fmt.Sprintf("%d", uid)
		}
	}

	// TODO: look up the actual GID
	gid2name := func(gid int) string {
		switch gid {
		case 0:
			return "root"
		default:
			return fmt.Sprintf("%d", gid)
		}
	}

	flags := flag.NewFlagSet("ls", flag.ContinueOnError)
	flags.SetOutput(virtOS.Stderr())
	flags.Usage = func() {
		fmt.Fprintln(virtOS.Stderr(), "Usage: ls [OPTION]... [FILE]...")
		fmt.Fprintln(virtOS.Stderr(), "List information about the FILEs (the current directory by default).")
	}

	listAll := flags.Bool("a", false, "don't ignore entries starting with .")
	longListing := flags.Bool("l", false, "use a long listing format")
	if err := flags.Parse(virtOS.Args()[1:]); err != nil {
		return 1
	}

	directoriesToList := flags.Args()
	if len(directoriesToList) == 0 {
		directoriesToList = append(directoriesToList, ".")
	}
	sort.Strings(directoriesToList)

	showDirectoryNames := len(directoriesToList) > 1

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
		fmt.Fprintf(virtOS.Stdout(), "total %d\n", totalSize)

		if *longListing {
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
				fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%d\t%s\t%s\n",
					f.Mode().String(),
					hardLinks,
					uid2name(uid),
					gid2name(gid),
					f.Size(),
					modTime,
					f.Name())
			}
			tw.Flush()
		} else {
			const minPaddingWidth = 2
			width := virtOS.GetPTY().Width
			if width == 0 {
				width = math.MaxInt32
			}
			maxCols := width / (longestNameLength + minPaddingWidth)

			var names []string
			for _, f := range paths {
				names = append(names, f.Name())
			}

			if maxCols == 0 || maxCols >= len(paths) {
				// Print all with two  spaces separated.
				fmt.Fprintln(virtOS.Stdout(), strings.Join(names, "  "))
			} else {
				cols := columnize(names, virtOS.GetPTY().Width)
				rows := len(names) / cols
				if len(names)%cols > 0 {
					rows++
				}

				tw := tabwriter.NewWriter(virtOS.Stdout(), 0, 0, 2, ' ', 0)
				for row := 0; row < rows; row++ {
					for col := 0; col < cols; col++ {
						index := (col * rows) + row
						entry := ""
						if index < len(names) {
							entry = names[index]
						}
						if col > 0 {
							fmt.Fprintf(tw, "\t")
						}
						fmt.Fprintf(tw, entry)
					}
					fmt.Fprintln(tw)
				}
				tw.Flush()
			}
		}
	}

	return exitCode
}

func columnize(names []string, screenWidth int) int {
	const colPadding = 2
	// 3 is the minimum column width, 1 char filename + 2 padding.
	columns := screenWidth / (1 + colPadding)
	for ; columns > 1; columns-- {
		maximums := make([]int, columns)
		total := (columns - 1) * colPadding
		rows := (len(names) / columns) + 1
		for i, name := range names {
			prevMax := maximums[i/rows]
			if nameLen := len(name); nameLen > prevMax {
				maximums[i/rows] = nameLen
				total = total - prevMax + nameLen
				if total > screenWidth {
					break
				}
			}
		}

		if total <= screenWidth {
			return columns
		}
	}

	return columns
}

func getUIDGID(fileInfo os.FileInfo) (uid, gid int) {
	switch v := (fileInfo.Sys()).(type) {
	case *syscall.Stat_t:
		return int(v.Uid), int(v.Gid)

	case tar.Header:
		return v.Uid, v.Gid

	case *tar.Header:
		return v.Uid, v.Gid

	default:
		// TODO: Log the type that caused the failure.
		return 0, 0
	}
}

var _ HoneypotCommandFunc = Ls

func init() {
	addBinCmd("ls", HoneypotCommandFunc(Ls))
}
