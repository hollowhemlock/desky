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
)

func main() {
	race := flag.Bool("race", false, "also run the race detector (requires a supported C toolchain)")
	flag.Parse()
	if err := verify(*race); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(race bool) error {
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
