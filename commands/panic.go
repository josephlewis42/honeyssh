package commands

import "josephlewis.net/osshit/core/vos"

func init() {
	addBinCmd("panic", func(_ vos.VOS) int {
		panic("some message")
	})
}
