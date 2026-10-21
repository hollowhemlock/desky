package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These focused tests also ensure -race-check links and executes a test binary
// when -run=^$ selects no tests. A package with no test files is not sufficient.
type fakeTools struct {
	settings                          goSettings
	library                           string
	compilerErr, supportErr, probeErr error
	probe                             []byte
	fail                              string
	calls                             []command
}

var toolFailure = errors.New("injected tool failure")

func TestMain(m *testing.M) {
	if os.Getenv("DESKY_VERIFY_RACE_PROBE") == "1" {
		fmt.Println(raceProbeMarker)
	}
	os.Exit(m.Run())
}

func fixtureVerifier(t *testing.T, goos string) (verifier, *fakeTools) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile("go.mod", []byte("module fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	library := filepath.Join(root, "libsynchronization.a")
	if err := os.WriteFile(library, nil, 0600); err != nil {
		t.Fatal(err)
	}
	f := &fakeTools{
		settings: goSettings{GOHOSTOS: goos, GOOS: goos, GOHOSTARCH: "amd64", GOARCH: "amd64", CC: "gcc", GOVERSION: "test-go"},
		library:  library,
		probe:    []byte("{\"Action\":\"output\",\"Package\":\"fixture/tools/verify\",\"Output\":\"desky: race probe binary started\\n\"}\n{\"Action\":\"pass\",\"Package\":\"fixture/tools/verify\"}\n"),
	}
	v := verifier{env: []string{"PATH=unchanged", "CGO_ENABLED=0", "GOOS=" + goos, "GOARCH=amd64"}, out: io.Discard, run: f.run}
	return v, f
}

func (f *fakeTools) run(c command) ([]byte, error) {
	f.calls = append(f.calls, c)
	if c.name+" "+strings.Join(c.args, " ") == f.fail {
		return []byte("tool diagnostic"), toolFailure
	}
	if c.name == "go" && c.args[0] == "env" {
		return json.Marshal(f.settings)
	}
	if c.name == "go" && c.args[0] == "list" {
		return []byte("runtime/race"), f.supportErr
	}
	if c.name == "go" && strings.Contains(strings.Join(c.args, " "), "-run=^$") {
		return f.probe, f.probeErr
	}
	if c.name != "go" && c.name != "gofmt" {
		if c.args[len(c.args)-1] == "libsynchronization.a" {
			return []byte(f.library), f.compilerErr
		}
		return []byte("fixture compiler version\n"), f.compilerErr
	}
	return nil, nil
}

func TestRaceReadinessFailures(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*fakeTools)
	}{
		{"target mismatch", "native target", func(f *fakeTools) { f.settings.GOARCH = "arm64" }},
		{"unsupported target", "race runtime", func(f *fakeTools) {
			f.settings.GOARCH, f.settings.GOHOSTARCH = "arm64", "arm64"
			f.supportErr = toolFailure
		}},
		{"missing compiler", "selected C compiler is unavailable", func(f *fakeTools) { f.compilerErr = os.ErrNotExist }},
		{"empty compiler", "empty compiler", func(f *fakeTools) { f.settings.CC = "" }},
		{"old runtime", "mingw-w64 runtime", func(f *fakeTools) { f.library = "libsynchronization.a" }},
		{"missing library", "mingw-w64 runtime", func(f *fakeTools) { f.library += ".missing" }},
		{"directory as library", "mingw-w64 runtime", func(f *fakeTools) { f.library = filepath.Dir(f.library) }},
		{"link failure", "failed to build or execute", func(f *fakeTools) { f.probeErr = toolFailure; f.probe = []byte("link failed") }},
		{"startup failure", "failed to build or execute", func(f *fakeTools) { f.probeErr = toolFailure; f.probe = []byte("DLL missing") }},
		{"no test files", "did not confirm execution", func(f *fakeTools) { f.probe = []byte("{\"Action\":\"skip\",\"Package\":\"fixture/tools/verify\"}\n") }},
		{"wrapper skipped binary", "did not confirm execution", func(f *fakeTools) { f.probe = []byte("{\"Action\":\"pass\",\"Package\":\"fixture/tools/verify\"}\n") }},
		{"no package completion", "did not confirm execution", func(f *fakeTools) {
			f.probe = []byte("{\"Action\":\"pass\",\"Package\":\"fixture/tools/verify\",\"Test\":\"SomeTest\"}\n")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, f := fixtureVerifier(t, "windows")
			tc.change(f)
			err := v.check([]string{"-race"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			for _, c := range f.calls {
				if c.name == "gofmt" || (c.name == "go" && c.args[0] == "vet") {
					t.Fatal("verification continued after failed prerequisites")
				}
			}
		})
	}
}

func TestRaceCompilerSelection(t *testing.T) {
	for _, goos := range []string{"windows", "linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			v, f := fixtureVerifier(t, goos)
			compiler := filepath.Join(t.TempDir(), "compiler with spaces", "gcc")
			f.settings.CC = `"` + compiler + `" --sysroot "root with spaces"`
			if goos == "darwin" {
				f.compilerErr = os.ErrNotExist // Go's own build/execute probe decides.
			}
			if err := v.check([]string{"-race-check"}); err != nil {
				t.Fatal(err)
			}
			compilerCalls := 0
			for _, c := range f.calls {
				if c.name == "go" {
					continue
				}
				compilerCalls++
				if c.name != compiler || !reflect.DeepEqual(c.args[:2], []string{"--sysroot", "root with spaces"}) {
					t.Fatalf("compiler arguments changed: %+v", c)
				}
			}
			want := map[string]int{"windows": 2, "linux": 1, "darwin": 0}[goos]
			if compilerCalls != want {
				t.Fatalf("compiler calls: %d, want %d", compilerCalls, want)
			}
			if _, err := os.Stat("bin"); !os.IsNotExist(err) {
				t.Fatal("readiness-only check created project build outputs")
			}
		})
	}
}

func TestVerificationEnvironmentIsolation(t *testing.T) {
	v, f := fixtureVerifier(t, "windows")
	v.env = append(v.env, "cgo_enabled=0", "CC=kept")
	original := append([]string(nil), v.env...)
	if err := v.check([]string{"-race", "-cross"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.env, original) {
		t.Fatal("changed caller environment")
	}
	crossCount, raceCount, ordinaryCount := 0, 0, 0
	for _, c := range f.calls {
		args := strings.Join(c.args, " ")
		switch {
		case strings.Contains(args, "desk-windows-") || strings.Contains(args, "desk-linux-") || strings.Contains(args, "desk-darwin-"):
			crossCount++
			assertEnvValue(t, c.env, "CGO_ENABLED", "0")
		case c.name == "gofmt" || args == "vet ./..." || args == "test ./..." || strings.HasSuffix(args, "./cmd/desk"):
			ordinaryCount++
			if !reflect.DeepEqual(c.env, original) {
				t.Fatalf("ordinary command changed environment: %s", args)
			}
		default:
			assertEnvValue(t, c.env, "CGO_ENABLED", "1")
			if args == "test -race -count=1 ./..." {
				raceCount++
			}
		}
		assertEnvValue(t, c.env, "PATH", "unchanged")
		assertEnvValue(t, c.env, "CC", "kept")
	}
	if crossCount != 6 || raceCount != 1 || ordinaryCount != 4 {
		t.Fatalf("cross/race/ordinary counts: %d/%d/%d", crossCount, raceCount, ordinaryCount)
	}
}

func assertEnvValue(t *testing.T, env []string, key, want string) {
	t.Helper()
	count := 0
	for _, value := range env {
		k, v, _ := strings.Cut(value, "=")
		if strings.EqualFold(k, key) {
			count++
			if v != want {
				t.Fatalf("%s=%q, want %q", key, v, want)
			}
		}
	}
	if count != 1 {
		t.Fatalf("%s appears %d times", key, count)
	}
}

func TestPlainVerificationNeedsNoCompiler(t *testing.T) {
	v, f := fixtureVerifier(t, "windows")
	f.compilerErr, f.supportErr = os.ErrNotExist, toolFailure
	if err := v.check(nil); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 4 {
		t.Fatalf("plain verification did extra prerequisite work: %+v", f.calls)
	}
}

func TestVerificationFailurePropagation(t *testing.T) {
	for _, fail := range []string{"go vet ./...", "go test ./...", "go test -race -count=1 ./...", "go build -o " + filepath.Join("bin", "desk-windows-amd64.exe") + " ./cmd/desk"} {
		t.Run(fail, func(t *testing.T) {
			v, f := fixtureVerifier(t, "linux")
			f.fail = fail
			if err := v.check([]string{"-race", "-cross"}); !errors.Is(err, toolFailure) {
				t.Fatalf("failure not propagated: %v", err)
			}
			last := f.calls[len(f.calls)-1]
			if last.name+" "+strings.Join(last.args, " ") != fail {
				t.Fatal("continued after failure")
			}
		})
	}
}

func TestStandaloneRaceCheck(t *testing.T) {
	for _, args := range [][]string{{"-race-check", "-race"}, {"-race-check", "-cross"}, {"unexpected"}} {
		v, f := fixtureVerifier(t, "linux")
		if err := v.check(args); err == nil || len(f.calls) != 0 {
			t.Fatalf("invalid arguments ran commands: %v", args)
		}
	}
}

func TestCompilerCommand(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  []string
	}{
		{`gcc -m64`, []string{"gcc", "-m64"}},
		{`"C:\compiler path\gcc.exe" '-DVALUE=a b'`, []string{`C:\compiler path\gcc.exe`, "-DVALUE=a b"}},
		{`ccache gcc --sysroot=/some/path`, []string{"ccache", "gcc", "--sysroot=/some/path"}},
		{`gcc $(literal) ""`, []string{"gcc", "$(literal)", ""}},
		{`gcc -DVALUE="literal"`, []string{"gcc", `-DVALUE="literal"`}},
		{`"unterminated`, nil}, {`"" gcc`, nil}, {" \t\n", nil},
	} {
		got, err := compilerCommand(tc.value)
		if tc.want == nil {
			if err == nil {
				t.Fatalf("accepted %q", tc.value)
			}
		} else if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q: got %q, %v; want %q", tc.value, got, err, tc.want)
		}
	}
}

func TestVerifierProcessHelper(t *testing.T) {
	switch os.Getenv("DESKY_VERIFY_HELPER") {
	case "fail":
		fmt.Fprint(os.Stderr, "intentional probe failure")
		os.Exit(23)
	case "wait":
		time.Sleep(30 * time.Second)
	}
}

func TestExecuteFailureAndTimeout(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	c := command{name: exe, args: []string{"-test.run=^TestVerifierProcessHelper$"}, env: replaceEnv(os.Environ(), "DESKY_VERIFY_HELPER=fail"), capture: true}
	output, err := execute(c)
	if err == nil || !bytes.Contains(output, []byte("intentional probe failure")) {
		t.Fatalf("lost process failure or diagnostics: %v, %s", err, output)
	}
	c.env = replaceEnv(os.Environ(), "DESKY_VERIFY_HELPER=wait")
	c.timeout = 100 * time.Millisecond
	if _, err := execute(c); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("missing timeout: %v", err)
	}
}
