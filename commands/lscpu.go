package commands

import (
	"fmt"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
)

var (
	// Match procfs.go
	lscpuText = strings.TrimSpace(`
Architecture:            x86_64
  CPU op-mode(s):        32-bit, 64-bit
  Address sizes:         40 bits physical, 48 bits virtual
  Byte Order:            Little Endian
CPU(s):                  1
  On-line CPU(s) list:   0
Vendor ID:               GenuineIntel
  BIOS Vendor ID:        unknown
  Model name:            unknown
    BIOS CPU family:     1
    CPU family:          6
    Model:               63
    Thread(s) per core:  1
    Core(s) per socket:  1
    Socket(s):           1
    Stepping:            2
    BogoMIPS:            1234.59
    Flags:               fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx
                         fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm pni pclmulqdq dtes64 monitor ds_cpl
                         vmx smx est tm2 ssse3 cx16 xtpr pdcm pcid dca sse4_1 sse4_2 x2apic popcnt tsc_deadline_timer
                         aes xsave avx xsaveopt
Virtualization features: 
  Virtualization:        VT-x
  Hypervisor vendor:     KVM
  Virtualization type:   full
Caches (sum of all):     
  L1d:                   32 KiB (1 instance)
  L1i:                   32 KiB (1 instance)
  L2:                    4 MiB (1 instance)
NUMA:                    
  NUMA node(s):          1
  NUMA node0 CPU(s):     0
`)
)

// Lscpu implements the lscpu command.
func Lscpu(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "lscpu [OPTION...]",
		Short: "Display information about the CPU architecture.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), lscpuText)
		return 0
	})
}

var _ vos.ProcessFunc = Lscpu

func init() {
	addBinCmd("lscpu", Lscpu)
}
