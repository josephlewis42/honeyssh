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

var _ HoneypotCommandFunc = Apt

func init() {
	addBinCmd("apt", HoneypotCommandFunc(Apt))
	addBinCmd("apt-get", HoneypotCommandFunc(Apt))
}

// '''apt-get fake
// suppports only the 'install PACKAGE' command.
// Places a 'Segfault' at /usr/bin/PACKAGE'''
// class command_aptget(HoneyPotCommand):
//     def start(self):
//         if len(self.args) > 0 and self.args[0] == 'install':
//             self.do_install()
//         else:
//             self.do_locked()
//
//     def sleep(self, time, time2 = None):
//         d = defer.Deferred()
//         if time2:
//             time = random.randint(time * 100, time2 * 100) / 100.0
//         reactor.callLater(time, d.callback, None)
//         return d
//
//     @inlineCallbacks
//     def do_install(self,*args):
//         if len(self.args) <= 1:
//             self.writeln('0 upgraded, 0 newly installed, 0 to remove and %s not upgraded.' % random.randint(200,300))
//             self.exit()
//             return
//
//         packages = {}
//         for y in [re.sub('[^A-Za-z0-9]', '', x) for x in self.args[1:]]:
//             packages[y] = {
//                 'version':      '%d.%d-%d' % \
//                     (random.choice((0, 1)),
//                     random.randint(1, 40),
//                     random.randint(1, 10)),
//                 'size':         random.randint(100, 900)
//                 }
//         totalsize = sum([packages[x]['size'] for x in packages])
//
//         self.writeln('Reading package lists... Done')
//         self.writeln('Building dependency tree')
//         self.writeln('Reading state information... Done')
//         self.writeln('The following NEW packages will be installed:')
//         self.writeln('  %s ' % ' '.join(packages))
//         self.writeln('0 upgraded, %d newly installed, 0 to remove and 259 not upgraded.' % \
//             len(packages))
//         self.writeln('Need to get %s.2kB of archives.' % (totalsize))
//         self.writeln('After this operation, %skB of additional disk space will be used.' % \
//             (totalsize * 2.2,))
//         i = 1
//         for p in packages:
//             self.writeln('Get:%d http://ftp.debian.org stable/main %s %s [%s.2kB]' % \
//                 (i, p, packages[p]['version'], packages[p]['size']))
//             i += 1
//             yield self.sleep(1, 2)
//         self.writeln('Fetched %s.2kB in 1s (4493B/s)''' % (totalsize))
//         self.writeln('Reading package fields... Done')
//         yield self.sleep(1, 2)
//         self.writeln('Reading package status... Done')
//         self.writeln('(Reading database ... 177887 files and directories currently installed.)')
//         yield self.sleep(1, 2)
//         for p in packages:
//             self.writeln('Unpacking %s (from .../archives/%s_%s_i386.deb) ...' % \
//                 (p, p, packages[p]['version']))
//             yield self.sleep(1, 2)
//         self.writeln('Processing triggers for man-db ...')
//         yield self.sleep(2)
//         for p in packages:
//             self.writeln('Setting up %s (%s) ...' % \
//                 (p, packages[p]['version']))
//             self.fs.mkfile('/usr/bin/%s' % p,
//                 0, 0, random.randint(10000, 90000), 33188)
//             self.honeypot.commands['/usr/bin/%s' % p] = \
//                 command_faked_package_class_factory.getCommand(p)
//             yield self.sleep(2)
//         self.exit()
//
//     def do_locked(self):
//         self.writeln('E: Could not open lock file /var/lib/apt/lists/lock - open (13: Permission denied)')
//         self.writeln('E: Unable to lock the list directory')
//         self.exit()
// commands['/usr/bin/apt-get'] = command_aptget
//
// # vim: set sw=4 et tw=0:
