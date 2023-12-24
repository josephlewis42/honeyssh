package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// No-op commands.
type NoOpCommand struct {
	Name     string
	Use      string
	Short    string
	Stdout   string
	ExitCode int
}

// Convert the no-op command description to a functioning command.
func (c *NoOpCommand) ToCommand() vos.ProcessFunc {
	return func(virtOS vos.VOS) int {
		cmd := &SimpleCommand{
			Use:   c.Use,
			Short: c.Short,
			// Never bail, even if args are bad.
			NeverBail: true,
		}

		return cmd.Run(virtOS, func() int {
			if c.Stdout != "" {
				w := virtOS.Stdout()
				fmt.Fprintln(w, c.Stdout)
			}

			return c.ExitCode
		})
	}
}

var noOpBinCommands = []NoOpCommand{
	{
		Name:  "kill",
		Use:   "kill [-s sigspec | -n signum | -sigspec] pid | jobspec ... or kill -l [sigspec]",
		Short: "Send a signal to a process.",
	},
	{
		Name:  "killall",
		Use:   "killall [OPTION]... [--] NAME...",
		Short: "Kill a process by name.",
	},
	{
		Name:  "lscpu",
		Use:   "lscpu [OPTION...]",
		Short: "Display information about the CPU architecture.",
		Stdout: mustDedent(`
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
            `),
	},
	{
		Name:  "lspci",
		Use:   "lspci [OPTION...]",
		Short: "List PCI devices.",
		Stdout: mustDedent(`
        00:00.0 Host bridge: Intel Corporation 440FX - 82441FX PMC [Natoma] (rev 02)
        00:01.0 ISA bridge: Intel Corporation 82371SB PIIX3 ISA [Natoma/Triton II]
        00:01.1 IDE interface: Intel Corporation 82371SB PIIX3 IDE [Natoma/Triton II]
        00:01.2 USB controller: Intel Corporation 82371SB PIIX3 USB [Natoma/Triton II] (rev 01)
        00:01.3 Bridge: Intel Corporation 82371AB/EB/MB PIIX4 ACPI (rev 03)
        00:02.0 VGA compatible controller: Red Hat, Inc. Virtio 1.0 GPU (rev 01)
        00:03.0 Ethernet controller: Red Hat, Inc. Virtio network device
        00:04.0 Ethernet controller: Red Hat, Inc. Virtio network device
        00:05.0 SCSI storage controller: Red Hat, Inc. Virtio SCSI
        00:06.0 SCSI storage controller: Red Hat, Inc. Virtio block device
        00:07.0 SCSI storage controller: Red Hat, Inc. Virtio block device
        00:08.0 Unclassified device [00ff]: Red Hat, Inc. Virtio memory balloon`),
	},
	{
		Name:   "lsusb",
		Use:    "lsusb [OPTION...]",
		Short:  "List USB devices.",
		Stdout: "Bus 001 Device 001: ID 1d6b:0001 Linux Foundation 1.1 root hub",
	},
	{
		Name:     "make",
		Use:      "make [options] [target] ...",
		Short:    "Run a dependency graph of commands.",
		Stdout:   "make: *** No rule to make target. Stop.",
		ExitCode: 1,
	},
	{
		Name:  "nohup",
		Use:   "nohup COMMAND [ARG]...",
		Short: "Run COMMAND, ignoring hangup signals.",
	},
	{
		Name:     "perl",
		Use:      "perl [switches] [--] [programfile] [arguments]",
		Short:    "The Perl 5 language interpreter.",
		Stdout:   "Can't locate perl5db.pl: No such file or directory",
		ExitCode: 1,
	},
	{
		Name:     "php",
		Use:      "php [options] [-f] <file> [--] [args...]",
		Short:    "PHP Command Line Interface.",
		Stdout:   "PHP:  Error parsing php.ini on line 424",
		ExitCode: 1,
	},
	{
		Name:  "pkill",
		Use:   "pkill [OPTION]... PATTERN",
		Short: "Signal a process by pattern",
	},
	{
		Name:     "python",
		Use:      "python [option] ... [-c cmd | -m mod | file | -] [arg] ...",
		Short:    "Embedded version of the Python language.",
		Stdout:   "python: No module named os",
		ExitCode: 1,
	},
	{
		Name:     "python3",
		Use:      "python3 [option] ... [-c cmd | -m mod | file | -] [arg] ...",
		Short:    "Embedded version of the Python language.",
		Stdout:   "python: No module named os",
		ExitCode: 1,
	},
	{
		Name:  "screen",
		Use:   "screen [-opts] [cmd [args]]",
		Short: "screen manager with VT100/ANSI terminal emulation",
	},
}

var noOpSbinCommands = []NoOpCommand{}

func init() {
	for i := range noOpBinCommands {
		cmd := noOpBinCommands[i]
		mustAddBinCmd(cmd.Name, cmd.ToCommand())
	}

	for i := range noOpSbinCommands {
		cmd := noOpSbinCommands[i]

		mustAddSbinCmd(cmd.Name, cmd.ToCommand())
	}
}
