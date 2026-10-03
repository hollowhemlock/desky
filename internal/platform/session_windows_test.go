package platform

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestInvisibleWindowStation(t *testing.T) {
	if visibleWindowStation(0) {
		t.Fatal("accepted absent station")
	}
	name, err := windows.UTF16PtrFromString(fmt.Sprintf("desky-test-%d-%d", os.Getpid(), time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	// Create a separate noninteractive station, never switch the process to it.
	station, _, err := user32.NewProc("CreateWindowStationW").Call(uintptr(unsafe.Pointer(name)), 0, 2, 0)
	if station == 0 {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			t.Skipf("host cannot create a noninteractive window station: %v", err)
		}
		t.Fatal("create noninteractive fixture:", err)
	}
	defer user32.NewProc("CloseWindowStation").Call(station)
	if visibleWindowStation(station) {
		t.Fatal("accepted absent desktop")
	}
}
