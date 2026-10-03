package platform

import (
	"os"
	"syscall"
)

func desktopSession() bool {
	// The active local console belongs to the logged-in graphical user.
	st, err := os.Stat("/dev/console")
	if err != nil || os.Getuid() == 0 {
		return false
	}
	native, ok := st.Sys().(*syscall.Stat_t)
	return ok && native.Uid == uint32(os.Getuid())
}
