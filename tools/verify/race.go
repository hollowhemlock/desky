package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type goSettings struct {
	GOHOSTOS, GOHOSTARCH, GOOS, GOARCH, CC, GOVERSION string
}

const raceProbeMarker = "desky: race probe binary started"

func (v verifier) raceReady() error {
	env := replaceEnv(v.env, "CGO_ENABLED=1")
	output, err := v.run(command{name: "go", args: []string{"env", "-json", "GOHOSTOS", "GOHOSTARCH", "GOOS", "GOARCH", "CC", "GOVERSION"}, env: env, capture: true, timeout: time.Minute})
	if err != nil {
		return fmt.Errorf("inspect Go race settings: %w\n%s", err, output)
	}
	var settings goSettings
	if err := json.Unmarshal(output, &settings); err != nil {
		return fmt.Errorf("decode Go settings: %w", err)
	}
	if settings.GOHOSTOS == "" || settings.GOHOSTARCH == "" || settings.GOOS == "" || settings.GOARCH == "" {
		return fmt.Errorf("Go did not report its host and target; check the Go installation")
	}
	if settings.GOOS != settings.GOHOSTOS || settings.GOARCH != settings.GOHOSTARCH {
		return fmt.Errorf("race checks require a native target: host %s/%s, target %s/%s; clear GOOS/GOARCH cross-compilation overrides", settings.GOHOSTOS, settings.GOHOSTARCH, settings.GOOS, settings.GOARCH)
	}
	output, err = v.run(command{name: "go", args: []string{"list", "-race", "runtime/race"}, env: env, capture: true, timeout: time.Minute})
	if err != nil {
		return fmt.Errorf("Go could not load the race runtime for %s/%s; check detector support and the Go installation: %w\n%s", settings.GOOS, settings.GOARCH, err, output)
	}
	fmt.Fprintf(v.out, "Race setup: %s %s/%s; CGO_ENABLED=1 for race commands only\n", settings.GOVERSION, settings.GOOS, settings.GOARCH)
	if settings.GOOS != "darwin" {
		compiler, err := compilerCommand(settings.CC)
		if err != nil {
			return fmt.Errorf("invalid Go CC setting: %w; quote compiler paths containing spaces", err)
		}
		output, err = v.run(command{name: compiler[0], args: append(compiler[1:], "--version"), env: env, capture: true, timeout: 30 * time.Second})
		if err != nil {
			return fmt.Errorf("selected C compiler is unavailable; install/configure a native compiler (Windows: MSYS2 UCRT64 GCC) and set CC/PATH for this shell; see README.md: %w\n%s", err, output)
		}
		version, _, _ := strings.Cut(strings.TrimSpace(string(output)), "\n")
		fmt.Fprintln(v.out, "C compiler:", version)
		if settings.GOOS == "windows" {
			output, err = v.run(command{name: compiler[0], args: append(compiler[1:], "--print-file-name", "libsynchronization.a"), env: env, capture: true, timeout: 30 * time.Second})
			library := strings.TrimSpace(string(output))
			info, statErr := os.Stat(library)
			if err != nil || !filepath.IsAbs(library) || statErr != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("Windows race checks require mingw-w64 runtime 8 or later: the selected compiler did not resolve an existing absolute libsynchronization.a path; install MSYS2 UCRT64 GCC and check CC/PATH (see README.md); compiler error: %v\n%s", err, output)
			}
		}
	}
	// JSON distinguishes an executed test binary from a successful package build
	// reporting [no test files]. The focused runner tests supply that binary.
	output, err = v.run(command{name: "go", args: []string{"test", "-race", "-count=1", "-run=^$", "-json", "./tools/verify"}, env: replaceEnv(env, "DESKY_VERIFY_RACE_PROBE=1"), capture: true, timeout: 5 * time.Minute})
	if err != nil {
		return fmt.Errorf("race probe failed to build or execute; check the compiler, runtime libraries and SDK diagnostics below (README.md has setup steps): %w\n%s", err, output)
	}
	if !probeExecuted(output) {
		return fmt.Errorf("race probe did not confirm execution of a test binary; tools/verify must contain _test.go files, and custom Go execution wrappers must actually run it\n%s", output)
	}
	fmt.Fprintln(v.out, "Race readiness passed: instrumented test binary built and executed; project tests were not run.")
	return nil
}

func probeExecuted(output []byte) bool {
	started, passed := false, false
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		var event struct{ Action, Package, Test, Output string }
		if json.Unmarshal(line, &event) != nil || event.Package == "" {
			continue
		}
		if event.Action == "output" && strings.TrimSpace(event.Output) == raceProbeMarker {
			started = true
		}
		if event.Action == "pass" && event.Test == "" {
			passed = true
		}
	}
	return started && passed
}

// Match Go's CC field syntax: whitespace separates fields, a quote at the start
// of a field quotes that field, and backslashes are literal (not shell escapes).
func compilerCommand(value string) ([]string, error) {
	var fields []string
	for value = strings.TrimLeft(value, " \t\r\n"); value != ""; value = strings.TrimLeft(value, " \t\r\n") {
		if value[0] == '\'' || value[0] == '"' {
			end := strings.IndexByte(value[1:], value[0])
			if end < 0 {
				return nil, fmt.Errorf("unterminated compiler quote")
			}
			fields = append(fields, value[1:end+1])
			value = value[end+2:]
		} else {
			end := strings.IndexAny(value, " \t\r\n")
			if end < 0 {
				end = len(value)
			}
			fields = append(fields, value[:end])
			value = value[end:]
		}
	}
	if len(fields) == 0 || fields[0] == "" {
		return nil, fmt.Errorf("empty compiler executable")
	}
	return fields, nil
}
