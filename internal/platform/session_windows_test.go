package platform

import (
	"errors"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestInvisibleWindowStation(t *testing.T) {
	if visibleWindowStation(0) {
		t.Fatal("accepted absent station")
	}
	current, _, err := getProcessWindowStation.Call()
	if current == 0 {
		t.Fatal("get current station:", err)
	}
	// Custom names require administrator rights. Let Windows choose the name
	// for this logon session; only read attributes and never switch stations.
	const winstaReadAttributes = 0x0002
	station, _, err := user32.NewProc("CreateWindowStationW").Call(0, 0, winstaReadAttributes, 0)
	if station == 0 {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			t.Skipf("host cannot create a noninteractive window station: %v", err)
		}
		t.Fatal("create noninteractive fixture:", err)
	}
	defer user32.NewProc("CloseWindowStation").Call(station)
	after, _, err := getProcessWindowStation.Call()
	if after != current {
		t.Fatal("fixture changed the process window station:", err)
	}
	// A failed attribute query also returns false from visibleWindowStation;
	// prove that the fixture is readable and actually noninteractive first.
	var flags struct{ inherit, reserved, flags uint32 }
	ok, _, err := getUserObjectInformation.Call(station, 1, uintptr(unsafe.Pointer(&flags)), unsafe.Sizeof(flags), 0)
	if ok == 0 {
		t.Fatal("read fixture station flags:", err)
	}
	if flags.flags&1 != 0 {
		t.Fatal("fixture station is visible")
	}
	if visibleWindowStation(station) {
		t.Fatal("accepted noninteractive station")
	}
}
