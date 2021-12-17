package commands

import (
	"fmt"
	"time"

	"josephlewis.net/osshit/core/vos"
)

// Uptime implements the UNIX uptime command.
func Uptime(virtOS vos.VOS) int {
	fmt.Fprintln(virtOS.Stdout(), formatUptime(virtOS))

	return 0
}

type uptimeable interface {
	Now() time.Time
	BootTime() time.Time
}

func formatUptime(virtOS uptimeable) string {
	now := virtOS.Now()
	uptime := virtOS.BootTime().Sub(now)
	day := (24 * time.Hour)
	uptimeDays := uptime / day
	uptime -= uptimeDays * day
	uptimeHours := uptime / time.Hour
	uptime -= uptimeHours * time.Hour
	uptimeMins := uptime / time.Minute

	return fmt.Sprintf(
		"%s up %d days,  %02d:%02d,  1 user,  load average: 0.08, 0.02, 0.01",
		now.Format("15:04:05"),
		uptimeDays,
		uptimeHours,
		uptimeMins,
	)
}

var _ vos.ProcessFunc = Uptime

func init() {
	addBinCmd("uptime", Uptime)
}
