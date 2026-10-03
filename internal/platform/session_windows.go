package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var getProcessWindowStation = user32.NewProc("GetProcessWindowStation")
var getUserObjectInformation = user32.NewProc("GetUserObjectInformationW")

func desktopSession() bool {
	station, _, _ := getProcessWindowStation.Call()
	return visibleWindowStation(station)
}

func visibleWindowStation(station uintptr) bool {
	if station == 0 {
		return false
	}
	// USEROBJECTFLAGS (two BOOLs and a DWORD), queried with UOI_FLAGS.
	var flags struct{ inherit, reserved, flags uint32 }
	ok, _, _ := getUserObjectInformation.Call(station, 1, uintptr(unsafe.Pointer(&flags)), unsafe.Sizeof(flags), 0)
	return ok != 0 && flags.flags&1 != 0 // WSF_VISIBLE
}
