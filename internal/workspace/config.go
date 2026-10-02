package workspace

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/hollowhemlock/desky/internal/fileio"
)

type Locations struct {
	Home        string
	ConfigFile  string
	PersonalDir string
	StateDir    string
}

func DefaultLocations() (Locations, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Locations{}, err
	}
	return locationsFor(runtime.GOOS, home, os.Getenv)
}

func locationsFor(platform, home string, env func(string) string) (Locations, error) {
	l := Locations{Home: home}
	base := ""
	switch platform {
	case "windows":
		base = env("LOCALAPPDATA")
		if !filepath.IsAbs(base) {
			return l, fmt.Errorf("LOCALAPPDATA must be an absolute path")
		}
		base = filepath.Join(base, "desky")
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "desky")
	case "linux":
		get := func(key, fallback string) (string, error) {
			v := env(key)
			if v == "" {
				return filepath.Join(home, fallback), nil
			}
			if !filepath.IsAbs(v) {
				return "", fmt.Errorf("%s must be an absolute path", key)
			}
			return v, nil
		}
		c, err := get("XDG_CONFIG_HOME", ".config")
		if err != nil {
			return l, err
		}
		d, err := get("XDG_DATA_HOME", ".local/share")
		if err != nil {
			return l, err
		}
		s, err := get("XDG_STATE_HOME", ".local/state")
		if err != nil {
			return l, err
		}
		l.ConfigFile = filepath.Join(c, "desky", "config.toml")
		l.PersonalDir = filepath.Join(d, "desky", "personal")
		l.StateDir = filepath.Join(s, "desky")
		return l, nil
	default:
		return l, Failure(7, "unsupported_platform", "supported platforms are macOS, Windows and Linux")
	}
	l.ConfigFile = filepath.Join(base, "config.toml")
	l.PersonalDir = filepath.Join(base, "personal")
	l.StateDir = filepath.Join(base, "device")
	return l, nil
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(path, `~\`)) {
		return filepath.Join(home, path[2:])
	}
	return path
}

func configError(path, field, reason string) *Error {
	return Failure(5, "invalid_configuration", fmt.Sprintf("%s: %s %s", path, field, reason))
}

func decodeTOML(path string, dst any) (toml.MetaData, error) {
	data, err := fileio.ReadRegular(path)
	if err != nil {
		return toml.MetaData{}, err
	}
	meta, err := toml.Decode(string(data), dst)
	if err != nil {
		return meta, configError(path, "TOML", "is malformed or has invalid field types")
	}
	if len(meta.Undecoded()) != 0 {
		return meta, configError(path, meta.Undecoded()[0].String(), "is unknown")
	}
	return meta, nil
}

func loadDefinition(path, root string) (Definition, error) {
	var d Definition
	_, err := decodeTOML(path, &d)
	if err != nil {
		return d, err
	}
	bad := func(field, why string) (Definition, error) { return d, configError(path, field, why) }
	if d.SchemaVersion != 1 {
		return bad("schema_version", "must be 1")
	}
	if d.Workspace == nil {
		return bad("workspace", "is required")
	}
	if d.Workspace.ID != nil {
		if !uuidPattern.MatchString(*d.Workspace.ID) {
			return bad("workspace.id", "must be a UUID v4")
		}
		*d.Workspace.ID = strings.ToLower(*d.Workspace.ID)
	}
	if d.Workspace.Name != nil && !validName(*d.Workspace.Name) {
		return bad("workspace.name", "must not be empty")
	}
	seen := map[string]bool{}
	if d.Resources == nil {
		d.Resources = []Resource{}
	}
	for i := range d.Resources {
		r := &d.Resources[i]
		r.Origin = "shared"
		field := fmt.Sprintf("resource[%d]", i)
		if !labelPattern.MatchString(r.ID) || seen[r.ID] {
			return bad(field+".id", "must be unique ASCII letters/digits/hyphens")
		}
		seen[r.ID] = true
		if r.Name != nil && !validName(*r.Name) {
			return bad(field+".name", "must not be empty")
		}
		switch r.Type {
		case "url":
			if r.Path != nil || r.Profile != nil || r.URL == nil || !validURL(*r.URL) {
				return bad(field, "requires only a valid HTTP(S) URL")
			}
		case "editor", "terminal":
			if r.URL != nil || r.Profile != nil {
				return bad(field, "accepts only a path")
			}
			if r.Path == nil {
				p := "."
				r.Path = &p
			}
			if *r.Path == "" || filepath.IsAbs(*r.Path) || filepath.VolumeName(*r.Path) != "" {
				return bad(field+".path", "must be relative")
			}
			target, err := filepath.EvalSymlinks(filepath.Join(root, *r.Path))
			if err != nil || !within(root, target) {
				return bad(field+".path", "must exist inside the checkout")
			}
			st, err := os.Stat(target)
			if err != nil || (!st.IsDir() && (r.Type == "terminal" || !st.Mode().IsRegular())) {
				return bad(field+".path", "has an unsupported file type")
			}
		case "app":
			if r.Path != nil || r.URL != nil || r.Profile == nil || !labelPattern.MatchString(*r.Profile) {
				return bad(field, "requires only a valid app profile name")
			}
		default:
			return bad(field+".type", "is unsupported")
		}
	}
	return d, nil
}

func validURL(raw string) bool {
	if strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) && u.Hostname() != "" && u.User == nil
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// Resolve existing ancestors as well, so a not-yet-created child of a symlink
// cannot evade the data-scope overlap checks.
func physicalPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	resolved, err = physicalPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}

type Profile struct {
	Executable string   `toml:"executable" json:"executable"`
	Args       []string `toml:"args" json:"args"`
}
type DeviceConfig struct {
	SchemaVersion   int                `toml:"schema_version" json:"schema_version"`
	PersonalDataDir string             `toml:"personal_data_dir" json:"personal_data_dir"`
	Launchers       map[string]Profile `toml:"launchers" json:"launchers"`
	Apps            map[string]Profile `toml:"apps" json:"apps"`
}

func LoadDevice(l Locations) (DeviceConfig, error) {
	d := DeviceConfig{SchemaVersion: 1, PersonalDataDir: l.PersonalDir, Launchers: map[string]Profile{}, Apps: map[string]Profile{}}
	var raw DeviceConfig
	meta, err := decodeTOML(l.ConfigFile, &raw)
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	if raw.SchemaVersion != 1 {
		return d, configError(l.ConfigFile, "schema_version", "must be 1")
	}
	if meta.IsDefined("personal_data_dir") {
		d.PersonalDataDir = expandHome(raw.PersonalDataDir, l.Home)
		if !filepath.IsAbs(d.PersonalDataDir) {
			return d, configError(l.ConfigFile, "personal_data_dir", "must be absolute")
		}
	}
	if raw.Launchers != nil {
		d.Launchers = raw.Launchers
	}
	if raw.Apps != nil {
		d.Apps = raw.Apps
	}
	for key, p := range d.Launchers {
		if key != "editor" && key != "terminal" {
			return d, configError(l.ConfigFile, "launchers."+key, "is unknown")
		}
		if err := validateProfile(p, true); err != nil {
			return d, configError(l.ConfigFile, "launchers."+key, err.Error())
		}
	}
	for key, p := range d.Apps {
		if !labelPattern.MatchString(key) {
			return d, configError(l.ConfigFile, "apps", "has an invalid profile name")
		}
		if err := validateProfile(p, false); err != nil {
			return d, configError(l.ConfigFile, "apps."+key, err.Error())
		}
	}
	return d, nil
}

func validateProfile(p Profile, target bool) error {
	if !filepath.IsAbs(p.Executable) {
		return fmt.Errorf("requires an absolute executable path")
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(p.Executable), ".exe") {
		return fmt.Errorf("requires a native .exe executable on Windows")
	}
	n := 0
	for _, arg := range p.Args {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("contains a NUL argument")
		}
		if strings.Contains(arg, "{path}") {
			if arg != "{path}" || !target {
				return fmt.Errorf("allows {path} only as a full editor/terminal argument")
			}
			n++
		}
	}
	if target && n != 1 {
		return fmt.Errorf("requires exactly one {path} argument")
	}
	return nil
}

func checkScopes(l Locations, personal, root string) error {
	p, err := physicalPath(personal)
	if err != nil {
		return stateError(err)
	}
	s, err := physicalPath(l.StateDir)
	if err != nil {
		return stateError(err)
	}
	if within(s, p) || within(p, s) || (root != "" && (within(root, p) || within(p, root) || within(root, s) || within(s, root))) {
		return configError(l.ConfigFile, "storage paths", "must not overlap the checkout or each other")
	}
	return nil
}
