// Package workspace owns workspace configuration, identity and the device registry.
package workspace

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	Exit    int    `json:"-"`
}

func (e *Error) Error() string { return e.Message }
func Failure(exit int, code, message string) *Error {
	return &Error{Code: code, Message: message, Exit: exit}
}
func stateError(err error) *Error {
	return Failure(9, "state_failure", fmt.Sprintf("device state: %v; preserve files and inspect registry.json.bak for recovery", err))
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var labelPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// NewID allocates an identity for an immutable personal record.
func NewID() (string, error)     { return newID() }
func ValidID(id string) bool     { return uuidPattern.MatchString(id) && id == strings.ToLower(id) }
func validName(name string) bool { return strings.TrimSpace(name) != "" }

type Checkout struct {
	CheckoutID     string     `json:"checkout_id"`
	RootPath       string     `json:"root_path"`
	WorkspaceID    string     `json:"workspace_id"`
	IdentitySource string     `json:"identity_source"`
	Name           string     `json:"name"`
	LastEnteredAt  *time.Time `json:"last_entered_at"`
	EntryCount     uint64     `json:"entry_count"`
}
type Resource struct {
	Origin  string  `toml:"-" json:"origin"`
	ID      string  `toml:"id" json:"id"`
	Type    string  `toml:"type" json:"type"`
	Name    *string `toml:"name,omitempty" json:"name,omitempty"`
	Path    *string `toml:"path,omitempty" json:"path,omitempty"`
	URL     *string `toml:"url,omitempty" json:"url,omitempty"`
	Profile *string `toml:"profile,omitempty" json:"profile,omitempty"`
}
type Definition struct {
	SchemaVersion int `toml:"schema_version"`
	Workspace     *struct {
		ID   *string `toml:"id,omitempty"`
		Name *string `toml:"name,omitempty"`
	} `toml:"workspace"`
	Resources []Resource `toml:"resource,omitempty"`
}
type Info struct {
	Checkout
	PreviousWorkspaceID string     `json:"previous_workspace_id,omitempty"`
	Registered          bool       `json:"registered"`
	Available           bool       `json:"available"`
	ConfigPath          string     `json:"config_path"`
	PersonalDataDir     string     `json:"personal_data_dir"`
	DeviceStateDir      string     `json:"device_state_dir"`
	Resources           []Resource `json:"resources"`
	Trust               string     `json:"trust"`
	ResourceWarnings    []string   `json:"resource_warnings,omitempty"`
}
type ListedCheckout struct {
	Checkout
	Available bool `json:"available"`
}
