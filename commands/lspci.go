package commands

import (
	"fmt"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
)

var (
	lspciText = strings.TrimSpace(`
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
00:08.0 Unclassified device [00ff]: Red Hat, Inc. Virtio memory balloon
  `)
)

// Lspci implements the lspci command.
func Lspci(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "lspci [OPTION...]",
		Short: "List PCI devices.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), lspciText)
		return 0
	})
}

var _ vos.ProcessFunc = Lspci

func init() {
	addBinCmd("lspci", Lspci)
}
