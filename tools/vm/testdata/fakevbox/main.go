// fakevbox is a test executable, never used to manage a real VM.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	root := os.Getenv("DESKY_FAKE_VBOX_ROOT")
	if root == "" {
		os.Exit(99)
	}
	args := os.Args[1:]
	log, err := os.OpenFile(filepath.Join(root, "calls.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	// Only argument paths are logged; password files are never read.
	_ = json.NewEncoder(log).Encode(args)
	_ = log.Close()
	if len(args) == 0 {
		os.Exit(2)
	}
	fail, _ := os.ReadFile(filepath.Join(root, "fail"))
	failure := strings.TrimSpace(string(fail))
	if failure == args[0] {
		os.Exit(42)
	}
	state := map[string]string{}
	data, _ := os.ReadFile(filepath.Join(root, "vm.json"))
	_ = json.Unmarshal(data, &state)
	flag := func(name string) string {
		for i, arg := range args {
			if arg == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	switch args[0] {
	case "--version":
		fmt.Println("7.2.20r175154")
	case "list":
		if len(args) != 2 || args[1] != "vms" {
			os.Exit(3)
		}
		if state["UUID"] != "" {
			fmt.Printf("%s {%s}\n", strconv.Quote(state["name"]), state["UUID"])
		}
	case "showvminfo":
		for key, value := range state {
			fmt.Printf("%s=%s\n", key, strconv.Quote(value))
		}
	case "startvm":
		state["VMState"] = "running"
		data, _ = json.Marshal(state)
		_ = os.WriteFile(filepath.Join(root, "vm.json"), data, 0600)
	case "guestcontrol":
		if len(args) < 3 || args[2] != "run" {
			os.Exit(3)
		}
		for i, arg := range args {
			if arg != "--" {
				continue
			}
			guestArgs := args[i+1:]
			if len(guestArgs) > 0 && guestArgs[0] == flag("--exe") {
				os.Exit(43)
			}
			switch flag("--exe") {
			case "/usr/bin/python3":
				if len(guestArgs) == 2 && guestArgs[0] == "-c" && strings.Contains(guestArgs[1], "os.path.expanduser") {
					if failure == "live-session" {
						os.Exit(45)
					}
					if home, err := os.ReadFile(filepath.Join(root, "guest-home")); err == nil {
						fmt.Print(string(home))
					} else {
						fmt.Println("/home/dev")
					}
					return
				}
			case "/usr/bin/pgrep":
				if failure == "guest-additions" {
					os.Exit(44)
				}
				fmt.Println("123")
				return
			}
			_ = json.NewEncoder(os.Stdout).Encode(guestArgs)
		}
	default:
		// No VM creation, installation, media changes, account changes or shutdown.
		os.Exit(90)
	}
}
