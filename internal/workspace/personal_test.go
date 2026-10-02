package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRegistryValidationRetainsAliasDetection(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "original")
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(s.Locations.Home, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	other := i.Checkout
	other.CheckoutID, _ = newID()
	other.RootPath = alias
	r := registry{1, []Checkout{i.Checkout, other}}
	data, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(s.Locations.StateDir, "registry.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(); err == nil {
		t.Fatal("duplicate filesystem identity accepted")
	}
}

func TestMRUDeterministicTies(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "same")
	if err != nil {
		t.Fatal(err)
	}
	r := registry{1, []Checkout{}}
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	for _, label := range []string{"z", "b", "a", "unentered", "count"} {
		c := i.Checkout
		c.CheckoutID, _ = newID()
		c.RootPath = filepath.Join(s.Locations.Home, label)
		if err := os.Mkdir(c.RootPath, 0700); err != nil {
			t.Fatal(err)
		}
		c.LastEnteredAt, c.EntryCount = &now, 1
		if label == "z" {
			c.Name = "zebra"
		}
		if label == "unentered" {
			c.LastEnteredAt = nil
		}
		if label == "count" {
			c.EntryCount = 2
		}
		r.Checkouts = append(r.Checkouts, c)
	}
	if err := s.store.save(r); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, c := range list {
		got = append(got, filepath.Base(c.RootPath))
	}
	if !reflect.DeepEqual(got, []string{"count", "a", "b", "z", "unentered"}) {
		t.Fatal(got)
	}
}

func TestURLNormalization(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"HTTPS://EXAMPLE.com:443/a?Q=One#Two", "https://example.com/a?Q=One#Two"},
		{"http://[::1]:80/a", "http://[::1]/a"},
		{"https://example.com:444/a", "https://example.com:444/a"},
		{"javascript:alert(1)", ""}, {"https://user:password@example.com", ""},
		{"https:///a", ""}, {"https://example.com/\n", ""}, {"/relative", ""},
	} {
		got, err := NormalizeURL(tc.raw)
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Fatalf("%q: %q %v", tc.raw, got, err)
		}
	}
}
