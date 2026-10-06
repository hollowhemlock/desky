// fakevbox is a test executable, never used to manage a real VM.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		if len(args) < 3 {
			os.Exit(3)
		}
		if args[2] == "copyfrom" {
			if failure == "receipt-copy" || !slices.Contains(args, "--no-replace") {
				fmt.Fprintln(os.Stderr, "fixture-private-detail")
				os.Exit(42)
			}
			source, destination := args[len(args)-2], args[len(args)-1]
			if !strings.HasSuffix(source, "/publication.json") {
				os.Exit(53)
			}
			staging := filepath.Base(strings.TrimSuffix(source, "/publication.json"))
			data, err := os.ReadFile(filepath.Join(root, staging, "publication.json"))
			if err != nil {
				os.Exit(54)
			}
			target, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				os.Exit(55)
			}
			if _, err := target.Write(data); err != nil {
				panic(err)
			}
			if err := target.Close(); err != nil {
				panic(err)
			}
			return
		}
		if args[2] == "copyto" {
			// Match GuestPath::BuildDestinationPath: a directory operand needs
			// its trailing separator, even with --target-directory.
			if !strings.HasSuffix(flag("--target-directory"), "/") {
				fmt.Fprintln(os.Stderr, "Destination already exists and is a directory; fixture-private-detail")
				os.Exit(1)
			}
			source := args[len(args)-1]
			if failure == "copyto-"+filepath.Base(source) {
				fmt.Fprintln(os.Stderr, "fixture-private-detail")
				os.Exit(42)
			}
			data, err := os.ReadFile(source)
			if err != nil {
				os.Exit(47)
			}
			staging := guestPath(root, flag("--target-directory"))
			target, err := os.OpenFile(filepath.Join(staging, filepath.Base(source)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				os.Exit(48)
			}
			// Model the vendor bug: CreateNew succeeds, then NoReplace sees
			// that new file and skips the write while returning success.
			if slices.Contains(args, "--no-replace") || failure == "empty-copy-"+filepath.Base(source) {
				data = nil
			} else if failure == "corrupt-copy-"+filepath.Base(source) {
				data[0] ^= 0xff
			}
			if _, err := target.Write(data); err != nil {
				panic(err)
			}
			if err := target.Close(); err != nil {
				panic(err)
			}
			return
		}
		if args[2] != "run" {
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
				if len(guestArgs) >= 5 && guestArgs[0] == "-c" && strings.Contains(guestArgs[1], "operation,upload,name") {
					upload := guestPath(root, guestArgs[3])
					if guestArgs[2] == "prepare" {
						if err := os.Mkdir(upload, 0700); err != nil {
							os.Exit(56)
						}
						return
					}
					if guestArgs[2] != "verify" || len(guestArgs) != 7 {
						os.Exit(57)
					}
					candidate := filepath.Join(upload, guestArgs[4])
					data, err := os.ReadFile(candidate)
					if err != nil || strconv.Itoa(len(data)) != guestArgs[5] || fmt.Sprintf("%x", sha256.Sum256(data)) != guestArgs[6] {
						fmt.Fprintln(os.Stderr, "fixture-private-detail")
						os.Exit(58)
					}
					if err := os.Link(candidate, filepath.Join(filepath.Dir(upload), guestArgs[4])); err != nil {
						os.Exit(59)
					}
					if err := os.Remove(candidate); err != nil {
						panic(err)
					}
					if err := os.Remove(upload); err != nil {
						panic(err)
					}
					return
				}
				if len(guestArgs) == 3 && guestArgs[0] == "-c" && strings.Contains(guestArgs[1], "tempfile.mkdtemp") {
					if failure == "stage" {
						fmt.Fprintln(os.Stderr, "fixture-private-detail")
						os.Exit(42)
					}
					dir, err := os.MkdirTemp(root, ".incoming-")
					if err != nil {
						panic(err)
					}
					fmt.Println(strings.TrimRight(guestArgs[2], "/") + "/" + filepath.Base(dir))
					return
				}
				if len(guestArgs) == 6 && guestArgs[1] == "publish" && guestArgs[5] == "--receipt" {
					if failure == "publish" {
						fmt.Fprintln(os.Stderr, "fixture-private-detail")
						os.Exit(42)
					}
					staging := filepath.Join(root, filepath.Base(guestArgs[2]))
					// Real Linux tests own extraction/integrity. Here all three
					// copies must finish before publication can be acknowledged.
					for _, name := range []string{"source.tar", "manifest.json", "transport.py"} {
						if _, err := os.Stat(filepath.Join(staging, name)); err != nil {
							os.Exit(50)
						}
					}
					archive, _ := os.ReadFile(filepath.Join(staging, "source.tar"))
					if fmt.Sprintf("%x", sha256.Sum256(archive)) != guestArgs[4] {
						os.Exit(51)
					}
					data, _ := os.ReadFile(filepath.Join(staging, "manifest.json"))
					var manifest struct{ Revision string }
					if json.Unmarshal(data, &manifest) != nil || manifest.Revision == "" {
						os.Exit(52)
					}
					receipt := map[string]any{
						"schema": 1, "revision": manifest.Revision,
						"directory":      strings.TrimRight(guestArgs[3], "/") + "/" + manifest.Revision,
						"archive_sha256": guestArgs[4], "incoming": guestArgs[2],
					}
					if field, ok := strings.CutPrefix(failure, "receipt-wrong-"); ok {
						receipt[field] = "fixture-private-detail"
					}
					data, _ = json.Marshal(receipt)
					if failure == "receipt-invalid" {
						data = []byte("{fixture-private-detail")
					}
					if failure != "receipt-missing" {
						if err := os.WriteFile(filepath.Join(staging, "publication.json"), data, 0600); err != nil {
							panic(err)
						}
					}
					if failure == "publication-empty" {
						return
					}
					if failure == "publication-noisy" {
						fmt.Println("fixture-private-detail")
					}
					if failure == "publication-wrong" {
						fmt.Println("/unexpected/fixture-private-detail")
						return
					}
					fmt.Println(strings.TrimRight(guestArgs[3], "/") + "/" + manifest.Revision)
					return
				}
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

// Map only this workflow's staging namespace into the disposable fixture root.
func guestPath(root, path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ".incoming-") {
			return filepath.Join(append([]string{root}, parts[i:]...)...)
		}
	}
	os.Exit(49)
	return ""
}
