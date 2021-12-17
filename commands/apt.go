package commands

import (
	"fmt"
	"math/rand"
	"time"

	"josephlewis.net/osshit/core/vos"
)

// Apt implements a fake apt command.
func Apt(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "apt command [options]",
		Short: "Manage system packages.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		switch {
		case len(cmd.Flags().Args()) == 0:
			fmt.Fprintln(w, "0 upgraded, 0 newly installed, 0 to remove and 235 not upgraded.")
			return 1

		case cmd.Flags().Args()[0] == "install":
			if len(cmd.Flags().Args()) < 2 {
				fmt.Fprintln(w, "missing package name")
				return 1
			}
			type packageInfo struct {
				name    string
				version string
				size    int
			}

			var packages []packageInfo
			var totalSize int
			for _, packageName := range cmd.Flags().Args()[1:] {
				pkg := packageInfo{
					name: packageName,
					// TODO: These should follow Benford's law (with 0s) to look more
					// realistic.
					version: fmt.Sprintf("%d.%d.%d", rand.Intn(2), rand.Intn(30), rand.Intn(100)),
					size:    500*1024 + rand.Intn(2*1024*1024),
				}
				totalSize += pkg.size
				packages = append(packages, pkg)
			}

			fmt.Fprintln(w, "Reading package lists... Done")
			fmt.Fprintln(w, "Building dependency tree")
			fmt.Fprintln(w, "Reading state information... Done")
			fmt.Fprintln(w, "The following NEW packages will be installed:")
			for _, pkg := range packages {
				fmt.Fprintf(w, "  %s\n", pkg.name)
			}
			fmt.Fprintf(
				w,
				"0 upgraded, %d newly installed, 0 to remove and 235 not upgraded.\n",
				len(packages),
			)
			fmt.Fprintf(w, "Need to get %s of archives.\n", BytesToHuman(int64(totalSize)))
			fmt.Fprintf(w, "After this operation, %s of additional disk space will be used.\n",
				BytesToHuman(int64(totalSize*2)))
			for i, pkg := range packages {
				fmt.Fprintf(w, "Get:%d http://archive.ubuntu.com/ubuntu updates/universe amd64 %s [%s].\n",
					i+1,
					pkg.name,
					BytesToHuman(int64(pkg.size)),
				)
				time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
			}
			fmt.Fprintf(w, "Fetched %s.\n", BytesToHuman(int64(totalSize)))

			for _, pkg := range packages {
				fmt.Fprintf(w, "Selecting previously unselected package %s.\n", pkg.name)
				fmt.Fprintln(w, "(Reading database ... 423488 files and directories currently installed.).")
				fmt.Fprintf(w, "Preparing to unpack .../%s_%s.deb.\n", pkg.name, pkg.version)
				fmt.Fprintf(w, "Unpacking %s (%s) ...\n", pkg.name, pkg.version)
				time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
			}

		default:
			fmt.Fprintln(w, "Could not open lock file /var/lib/apt/lists/lock - open (13: Permission denied)")
			fmt.Fprintln(w, "Unable to lock the list directory")
			return 1
		}
		// Noop
		return 0
	})
}

var _ vos.ProcessFunc = Apt

func init() {
	addBinCmd("apt", Apt)
	addBinCmd("apt-get", Apt)
}
