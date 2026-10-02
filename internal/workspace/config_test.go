package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSharedConfigurationValidation(t *testing.T) {
	cases := map[string]string{
		"version":            "schema_version = 2\n[workspace]\n",
		"missing version":    "[workspace]\n",
		"missing workspace":  "schema_version = 1\n",
		"unknown":            "schema_version = 1\nextra = true\n[workspace]\n",
		"bad uuid":           "schema_version = 1\n[workspace]\nid = 'wrong'\n",
		"empty name":         "schema_version = 1\n[workspace]\nname = ''\n",
		"duplicate resource": "schema_version = 1\n[workspace]\n[[resource]]\nid='e'\ntype='editor'\n[[resource]]\nid='e'\ntype='editor'\n",
		"unsafe URL":         "schema_version = 1\n[workspace]\n[[resource]]\nid='u'\ntype='url'\nurl='javascript:alert(1)'\n",
		"URL credentials":    "schema_version = 1\n[workspace]\n[[resource]]\nid='u'\ntype='url'\nurl='https://name:secret@example.com'\n",
		"wrong type fields":  "schema_version = 1\n[workspace]\n[[resource]]\nid='e'\ntype='editor'\nurl='https://example.com'\n",
		"escape":             "schema_version = 1\n[workspace]\n[[resource]]\nid='e'\ntype='editor'\npath='..'\n",
		"missing target":     "schema_version = 1\n[workspace]\n[[resource]]\nid='e'\ntype='editor'\npath='absent'\n",
		"repo argv":          "schema_version = 1\n[workspace]\n[[resource]]\nid='e'\ntype='editor'\nargs=['danger']\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			s, root := fixture(t)
			put(t, filepath.Join(root, "workspace.toml"), body)
			_, err := s.Info(root, ".")
			code(t, err, "invalid_configuration")
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("diagnostic exposed field contents")
			}
		})
	}
	t.Run("optional fields and case normalization", func(t *testing.T) {
		s, root := fixture(t)
		put(t, filepath.Join(root, "workspace.toml"), "schema_version=1\n[workspace]\nid='225116AD-90E9-42CD-9770-65D4B5D467AB'\n")
		i, err := s.Info(root, ".")
		if err != nil {
			t.Fatal(err)
		}
		if i.Name != filepath.Base(root) || len(i.Resources) != 0 || i.WorkspaceID != strings.ToLower(i.WorkspaceID) {
			t.Fatal("defaults not applied")
		}
	})
}

func TestDeviceConfiguration(t *testing.T) {
	s, root := fixture(t)
	v, err := s.Config("personal_data_dir")
	if err != nil || v != s.Locations.PersonalDir {
		t.Fatalf("default: %v %v", v, err)
	}
	if _, err := os.Stat(filepath.Dir(s.Locations.ConfigFile)); !os.IsNotExist(err) {
		t.Fatal("config read created directories")
	}
	cases := []string{
		"schema_version=2\n",
		"schema_version=1\npersonal_data_dir='relative'\n",
		"schema_version=1\npersonal_data_dir=''\n",
		"schema_version=1\nunknown=true\n",
		"schema_version=1\n[launchers.editor]\nexecutable='relative'\nargs=['{path}']\n",
		"schema_version=1\n[apps.bad]\nexecutable='relative'\nargs=['{path}']\n",
	}
	for _, body := range cases {
		put(t, s.Locations.ConfigFile, body)
		_, err := s.Config("personal_data_dir")
		code(t, err, "invalid_configuration")
	}
	put(t, s.Locations.ConfigFile, "schema_version=1\npersonal_data_dir='~/saved'\n")
	v, err = s.Config("personal_data_dir")
	if err != nil || v != filepath.Join(s.Locations.Home, "saved") {
		t.Fatalf("home expansion: %v %v", v, err)
	}
	for _, overlap := range []string{s.Locations.StateDir, root, filepath.Dir(root)} {
		put(t, s.Locations.ConfigFile, fmt.Sprintf("schema_version=1\npersonal_data_dir='%s'\n", overlap))
		_, err = s.Info(root, ".")
		code(t, err, "invalid_configuration")
	}
	t.Run("symlink storage overlap", func(t *testing.T) {
		link := filepath.Join(s.Locations.Home, "alias")
		if err := os.Symlink(root, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		put(t, s.Locations.ConfigFile, fmt.Sprintf("schema_version=1\npersonal_data_dir='%s'\n", filepath.Join(link, "not-created")))
		_, err := s.Info(root, ".")
		code(t, err, "invalid_configuration")
	})
}

func TestProfileValidation(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "editor.exe")
	for _, p := range []Profile{{exe, []string{}}, {exe, []string{"prefix{path}"}}, {exe, []string{"{path}", "{path}"}}, {"relative", []string{"{path}"}}} {
		if validateProfile(p, true) == nil {
			t.Fatalf("invalid profile accepted: %+v", p)
		}
	}
	if err := validateProfile(Profile{exe, []string{"{path}"}}, true); err != nil {
		t.Fatal(err)
	}
	if err := validateProfile(Profile{exe, []string{"{path}"}}, false); err == nil {
		t.Fatal("app accepted substitution")
	}
	if runtime.GOOS == "windows" && validateProfile(Profile{exe + ".cmd", []string{"{path}"}}, true) == nil {
		t.Fatal("accepted batch launcher")
	}
}

func TestLocationDefaults(t *testing.T) {
	home := t.TempDir()
	env := func(string) string { return "" }
	l, err := locationsFor("linux", home, env)
	if err != nil {
		t.Fatal(err)
	}
	if l.StateDir != filepath.Join(home, ".local", "state", "desky") || l.PersonalDir == l.StateDir {
		t.Fatal("wrong Linux scopes")
	}
	l, err = locationsFor("darwin", home, env)
	if err != nil {
		t.Fatal(err)
	}
	if l.StateDir != filepath.Join(home, "Library", "Application Support", "desky", "device") {
		t.Fatal("wrong macOS scopes")
	}
	l, err = locationsFor("windows", home, func(string) string { return home })
	if err != nil {
		t.Fatal(err)
	}
	if l.StateDir != filepath.Join(home, "desky", "device") {
		t.Fatal("wrong Windows scopes")
	}
	if _, err := locationsFor("linux", home, func(string) string { return "relative" }); err == nil {
		t.Fatal("accepted relative XDG base")
	}
}

func TestCheckoutCannotBeInsideDeviceState(t *testing.T) {
	s, _ := fixture(t)
	root := filepath.Join(s.Locations.StateDir, "project")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	_, err := s.Init(root, "wrong scope")
	code(t, err, "invalid_configuration")
	if _, err := os.Stat(filepath.Join(root, "workspace.toml")); !os.IsNotExist(err) {
		t.Fatal("wrote config inside device state")
	}
}
