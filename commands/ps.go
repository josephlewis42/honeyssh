package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

var (
	psHeader = `  USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND`
	psSystem = `  root         1  2.7  0.9  21952  9868 ?        Ss   05:04   0:01 /sbin/init
  root         2  0.0  0.0      0     0 ?        S    05:04   0:00 [kthreadd]
  root         3  0.0  0.0      0     0 ?        I<   05:04   0:00 [rcu_gp]
  root         4  0.0  0.0      0     0 ?        I<   05:04   0:00 [rcu_par_gp]
  root         5  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/0:0-cgroup_destroy]
  root         6  0.0  0.0      0     0 ?        I<   05:04   0:00 [kworker/0:0H-kblockd]
  root         7  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/u4:0-events_unbound]
  root         8  0.0  0.0      0     0 ?        I<   05:04   0:00 [mm_percpu_wq]
  root         9  0.0  0.0      0     0 ?        S    05:04   0:00 [ksoftirqd/0]
  root        10  0.0  0.0      0     0 ?        I    05:04   0:00 [rcu_sched]
  root        11  0.0  0.0      0     0 ?        I    05:04   0:00 [rcu_bh]
  root        12  0.0  0.0      0     0 ?        S    05:04   0:00 [migration/0]
  root        13  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/0:1-events]
  root        14  0.0  0.0      0     0 ?        S    05:04   0:00 [cpuhp/0]
  root        15  0.0  0.0      0     0 ?        S    05:04   0:00 [cpuhp/1]
  root        16  0.8  0.0      0     0 ?        S    05:04   0:00 [migration/1]
  root        17  0.0  0.0      0     0 ?        S    05:04   0:00 [ksoftirqd/1]
  root        18  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/1:0-cgroup_destroy]
  root        19  0.0  0.0      0     0 ?        I<   05:04   0:00 [kworker/1:0H-kblockd]
  root        20  0.0  0.0      0     0 ?        S    05:04   0:00 [kdevtmpfs]
  root        21  0.0  0.0      0     0 ?        I<   05:04   0:00 [netns]
  root        22  0.0  0.0      0     0 ?        S    05:04   0:00 [kauditd]
  root        23  0.0  0.0      0     0 ?        S    05:04   0:00 [khungtaskd]
  root        24  0.0  0.0      0     0 ?        S    05:04   0:00 [oom_reaper]
  root        25  0.0  0.0      0     0 ?        I<   05:04   0:00 [writeback]
  root        26  0.0  0.0      0     0 ?        S    05:04   0:00 [kcompactd0]
  root        27  0.0  0.0      0     0 ?        SN   05:04   0:00 [ksmd]
  root        28  0.0  0.0      0     0 ?        SN   05:04   0:00 [khugepaged]
  root        29  0.0  0.0      0     0 ?        I<   05:04   0:00 [crypto]
  root        30  0.0  0.0      0     0 ?        I<   05:04   0:00 [kintegrityd]
  root        31  0.0  0.0      0     0 ?        I<   05:04   0:00 [kblockd]
  root        32  0.0  0.0      0     0 ?        S    05:04   0:00 [watchdogd]
  root        33  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/1:1-rcu_gp]
  root        34  0.0  0.0      0     0 ?        S    05:04   0:00 [kswapd0]
  root        50  0.0  0.0      0     0 ?        I<   05:04   0:00 [kthrotld]
  root        51  0.0  0.0      0     0 ?        I<   05:04   0:00 [ipv6_addrconf]
  root        52  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/u4:1-events_unbound]
  root        61  0.0  0.0      0     0 ?        I<   05:04   0:00 [kstrp]
  root        64  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/0:2-events]
  root       126  0.0  0.0      0     0 ?        S    05:04   0:00 [scsi_eh_0]
  root       127  0.0  0.0      0     0 ?        I<   05:04   0:00 [scsi_tmf_0]
  root       133  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/u4:2]
  root       159  0.0  0.0      0     0 ?        I<   05:04   0:00 [kworker/1:1H-kblockd]
  root       160  0.0  0.0      0     0 ?        I<   05:04   0:00 [kworker/0:1H-kblockd]
  root       161  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/1:2-mm_percpu_wq]
  root       189  0.0  0.0      0     0 ?        I<   05:04   0:00 [kworker/u5:0]
  root       191  0.0  0.0      0     0 ?        S    05:04   0:00 [jbd2/sda1-8]
  root       192  0.0  0.0      0     0 ?        I<   05:04   0:00 [ext4-rsv-conver]
  root       203  0.0  0.0      0     0 ?        S    05:04   0:00 [hwrng]
  root       226  0.3  0.7  30140  7904 ?        Ss   05:04   0:00 /lib/systemd/systemd-journald
  root       236  0.1  0.4  20208  4624 ?        Ss   05:04   0:00 /lib/systemd/systemd-udevd
  root       296  0.6  0.7   8084  7432 ?        Ss   05:04   0:00 /usr/sbin/haveged --Foreground --verbose=1 -w 1024
  message+   343  0.0  0.3   8700  3636 ?        Ss   05:04   0:00 /usr/bin/dbus-daemon
  root       371  0.2  1.6  28416 16808 ?        Ss   05:04   0:00 /usr/bin/unattended-upgrade-shutdown --wait-for-signal
  root       378  0.3  2.1 120960 22152 ?        Ssl  05:04   0:00 /usr/bin/google_osconfig_agent
  root       390  0.0  0.1   2648  1652 tty1     Ss+  05:04   0:00 /sbin/agetty -o -p -- \u --noclear tty1 linux
  root       393  0.0  0.5 225824  5636 ?        Ssl  05:04   0:00 /usr/sbin/rsyslogd -n -iNONE
  root       407  0.5  1.7 114304 17756 ?        Ssl  05:04   0:00 /usr/bin/google_guest_agent
  root       501  0.0  0.6  15852  6792 ?        Ss   05:04   0:00 /usr/sbin/sshd -D
  root       504  0.0  0.7  19392  7308 ?        Ss   05:04   0:00 /lib/systemd/systemd-logind
  root       508  0.0  0.2   7264  2664 ?        Ss   05:04   0:00 /usr/sbin/cron -f
  root       510  0.0  0.0      0     0 ?        I    05:04   0:00 [kworker/0:3-cgroup_destroy]
  root       554  0.2  0.7  16612  7904 ?        Ss   05:04   0:00 sshd: joehms22 [priv]
  root       561  0.1  0.8  21024  8532 ?        Ss   05:04   0:00 /lib/systemd/systemd --user
  root       562  0.0  0.2  22916  2376 ?        S    05:04   0:00 (sd-pam)
  root       575  0.0  0.4  16612  4780 ?        R    05:04   0:00 sshd`
	psUser = `  root       576  0.0  0.3   5752  3584 pts/0    Ss   05:04   0:00 sh
  root       581  0.0  0.3   9392  3060 pts/0    R+   05:05   0:00 ps`
)

// Ps implements a fake ps command.
func Ps(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "ps [options]",
		Short: "Report a snapshot of system processes.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	showAll := cmd.Flags().Bool('a', "show all")

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), psHeader)

		if *showAll {
			fmt.Fprintln(virtOS.Stdout(), psSystem)
		}

		fmt.Fprintln(virtOS.Stdout(), psUser)
		return 1
	})
}

var _ vos.ProcessFunc = Ps

func init() {
	addBinCmd("ps", Ps)
}
