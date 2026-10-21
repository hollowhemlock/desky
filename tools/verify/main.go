// Run from the repository root with go run ./tools/verify.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	v := verifier{env: os.Environ(), out: os.Stdout, run: execute}
	if err := v.check(os.Args[1:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type command struct {
	name    string
	args    []string
	env     []string
	capture bool
	timeout time.Duration
}

type verifier struct {
	env []string
	out io.Writer
	run func(command) ([]byte, error)
}

func execute(c command) ([]byte, error) {
	ctx := context.Background()
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Env = c.env
	cmd.WaitDelay = time.Second
	var out []byte
	var err error
	if c.capture {
		out, err = cmd.CombinedOutput()
	} else {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		err = cmd.Run()
	}
	if ctx.Err() != nil {
		return out, fmt.Errorf("%s timed out: %w", c.name, ctx.Err())
	}
	return out, err
}

func (v verifier) check(args []string) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(v.out)
	race := flags.Bool("race", false, "check prerequisites and also run uncached race tests")
	raceCheck := flags.Bool("race-check", false, "check race build/execution readiness without running project tests")
	cross := flags.Bool("cross", false, "also compile Windows/macOS/Linux for amd64 and arm64 (not native launch tests)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments; use -help")
	}
	if *raceCheck && (*race || *cross) {
		return fmt.Errorf("-race-check is standalone; do not combine it with -race or -cross")
	}
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run verification from the repository root")
	}
	if *race || *raceCheck {
		if err := v.raceReady(); err != nil {
			return err
		}
	}
	if *raceCheck {
		return nil
	}
	if err := v.verify(*race); err != nil {
		return err
	}
	if *cross {
		return v.crossBuild()
	}
	return nil
}

func (v verifier) verify(race bool) error {
	fmt.Fprintf(v.out, "Native verification: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
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
	out, err := v.run(command{name: "gofmt", args: append([]string{"-l"}, files...), env: v.env, capture: true})
	if err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, out)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		return fmt.Errorf("run gofmt on:\n%s", out)
	}
	commands := [][]string{{"vet", "./..."}, {"test", "./..."}}
	if race {
		commands = append(commands, []string{"test", "-race", "-count=1", "./..."})
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
		fmt.Fprintln(v.out, "go", args)
		env := v.env
		if args[0] == "test" && len(args) > 1 && args[1] == "-race" {
			env = replaceEnv(env, "CGO_ENABLED=1")
		}
		if _, err := v.run(command{name: "go", args: args, env: env}); err != nil {
			return fmt.Errorf("go %s failed: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

func (v verifier) crossBuild() error {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			name := "desk-" + goos + "-" + arch
			if goos == "windows" {
				name += ".exe"
			}
			args := []string{"build", "-o", filepath.Join("bin", name), "./cmd/desk"}
			env := replaceEnv(v.env, "GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0")
			fmt.Fprintf(v.out, "Compile only: %s/%s\n", goos, arch)
			if _, err := v.run(command{name: "go", args: args, env: env}); err != nil {
				return fmt.Errorf("cross-build %s/%s failed: %w", goos, arch, err)
			}
		}
	}
	return nil
}

func replaceEnv(env []string, overrides ...string) []string {
	result := make([]string, 0, len(env)+len(overrides))
	for _, value := range env {
		key, _, _ := strings.Cut(value, "=")
		replaced := false
		for _, override := range overrides {
			other, _, _ := strings.Cut(override, "=")
			if strings.EqualFold(key, other) {
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, value)
		}
	}
	return append(result, overrides...)
}
