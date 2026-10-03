// Run from the repository root with go run ./tools/verify.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	race := flag.Bool("race", false, "also run the race detector (requires a supported C toolchain)")
	cross := flag.Bool("cross", false, "also compile Windows/macOS/Linux for amd64 and arm64 (not native launch tests)")
	flag.Parse()
	if err := verify(*race); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *cross {
		if err := crossBuild(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func verify(race bool) error {
	fmt.Printf("Native verification: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run verification from the repository root")
	}
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".cache" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	out, err := exec.Command("gofmt", append([]string{"-l"}, files...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, out)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		return fmt.Errorf("run gofmt on:\n%s", out)
	}
	commands := [][]string{{"vet", "./..."}, {"test", "./..."}}
	if race {
		commands = append(commands, []string{"test", "-race", "./..."})
	}
	name := "desk"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	commands = append(commands, []string{"build", "-o", filepath.Join("bin", name), "./cmd/desk"})
	for _, args := range commands {
		fmt.Println("go", args)
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func crossBuild() error {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			name := "desk-" + goos + "-" + arch
			if goos == "windows" {
				name += ".exe"
			}
			cmd := exec.Command("go", "build", "-o", filepath.Join("bin", name), "./cmd/desk")
			for _, value := range os.Environ() {
				key, _, _ := strings.Cut(value, "=")
				if !strings.EqualFold(key, "GOOS") && !strings.EqualFold(key, "GOARCH") && !strings.EqualFold(key, "CGO_ENABLED") {
					cmd.Env = append(cmd.Env, value)
				}
			}
			cmd.Env = append(cmd.Env, "GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			fmt.Printf("Compile only: %s/%s\n", goos, arch)
			if err := cmd.Run(); err != nil {
				return err
			}
		}
	}
	return nil
}
