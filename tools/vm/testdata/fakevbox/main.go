// fakevbox is a test executable, never used to manage a real VM.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	// Password file names are permitted; password contents are never read or logged.
	_ = json.NewEncoder(log).Encode(args)
	_ = log.Close()
	if len(args) == 0 {
		os.Exit(2)
	}
	if fail, _ := os.ReadFile(filepath.Join(root, "fail")); len(fail) != 0 {
		match := strings.TrimSpace(string(fail))
		if match == args[0] || len(args) > 1 && match == strings.Join(args[:2], " ") {
			os.Exit(42)
		}
	}
	state := map[string]string{}
	data, _ := os.ReadFile(filepath.Join(root, "vm.json"))
	_ = json.Unmarshal(data, &state)
	flag := func(name string) string {
		for i, arg := range args {
			if arg == name && i+1 < len(args) {
				return args[i+1]
			}
			if strings.HasPrefix(arg, name+"=") {
				return strings.TrimPrefix(arg, name+"=")
			}
		}
		return ""
	}
	switch args[0] {
	case "--version":
		fmt.Println("7.2.20r175154")
	case "list":
		if args[1] == "vms" && state["UUID"] != "" {
			fmt.Printf("\"%s\" {%s}\n", state["name"], state["UUID"])
		}
		if args[1] == "systemproperties" {
			fmt.Println("Default machine folder: " + filepath.Join(root, "machines"))
		}
	case "showvminfo":
		for key, value := range state {
			fmt.Printf("%s=\"%s\"\n", key, strings.ReplaceAll(value, "\\", "\\\\"))
		}
	case "createvm":
		state = map[string]string{"UUID": flag("--uuid"), "name": flag("--name"), "VMState": "poweroff", "CfgFile": filepath.Join(flag("--basefolder"), flag("--name"), "machine.vbox")}
		_ = os.MkdirAll(filepath.Dir(state["CfgFile"]), 0700)
	case "createmedium":
		_ = os.WriteFile(flag("--filename"), []byte("fake disk"), 0600)
	case "showmediuminfo":
		if changed, _ := os.ReadFile(filepath.Join(root, "disk-identity")); len(changed) != 0 {
			fmt.Println("UUID: " + strings.TrimSpace(string(changed)))
		} else {
			fmt.Println("UUID: 5c46d971-6548-4d03-9a9b-11ad5d02b12e")
		}
	case "storagectl":
		state["storagecontrollername0"] = "SATA"
	case "storageattach":
		state[flag("--storagectl")+"-"+flag("--port")+"-0"] = flag("--medium")
	case "startvm":
		state["VMState"] = "running"
	case "guestcontrol":
		if args[2] != "run" {
			os.Exit(3)
		}
		for i, arg := range args {
			if arg == "--" {
				guestArgs := args[i+1:]
				if len(guestArgs) > 0 && guestArgs[0] == flag("--exe") {
					os.Exit(43)
				}
				_ = json.NewEncoder(os.Stdout).Encode(guestArgs)
			}
		}
	case "unattended":
		if args[1] == "detect" {
			fmt.Println("OSTypeId=\"Ubuntu24_LTS_64\"\nOSVersion=\"24.04.5\"\nIsInstallSupported=\"on\"")
		} else {
			state["VMState"] = "running"
		}
	case "modifyvm":
	default:
		os.Exit(3)
	}
	if state["UUID"] != "" {
		data, _ = json.Marshal(state)
		_ = os.WriteFile(filepath.Join(root, "vm.json"), data, 0600)
	}
}
