# Desky first-release specification

Status: selected first-release contracts; workspace metadata, Windows entry,
personal URL persistence and selection UI (increments 1-3) are implemented.
Native launch adapters for other platforms remain planned.
[README.md](README.md) describes the working surface. [plan.md](plan.md) owns product boundaries;
[namespace.md](namespace.md) supplies CLI intent and provisional examples. This
document owns the concrete first-release contract. [IMPLEMENTATION.md](IMPLEMENTATION.md)
owns sequencing and acceptance, not additional requirements.

## Release boundary and decisions

Deliver a local CLI that selects a known project or directory, opens its editor,
terminal, named desktop applications and project URLs, and remembers manually
saved URLs. No daemon, GUI, hosted backend, accounts, telemetry, environment
management, process supervision, teardown, or window ownership. Launch means
requesting an application/OS action, not proving that a workspace is now open.

Use `desk` provisionally as the executable and `desky` as the stable storage
directory name. Bare `desk` always selects recent workspaces; `desk .` opens the
current project. This refines the conflicting shorthand examples in the product
brief using the explicit primary interaction in namespace sections 2 and 27.
Command naming can change before release without migrating storage.

Include explicit `open`, initialization, list/info, manual URL management, and
configuration diagnostics. Defer aliases, clipboard access, general resource
editing, `recent`/`find` convenience commands, previous-workspace `-`, shell
hooks, browser capture/extensions, and optional actions/`run`. These are proposed
or optional in the sources, not first-release obligations. Archive and pin state
changes are included so a saved URL can stop opening without deleting it.

## Architecture and ownership

Choose Go, one module and one CLI executable, with the standard library for
arguments, JSON, paths, process invocation, hashing and tests. Use
[BurntSushi/toml](https://github.com/BurntSushi/toml) for TOML parsing and encoding;
use `golang.org/x/sys/windows` for the small Windows native boundary if required.
The supported Go release and dependency versions are pinned in
[go.mod](go.mod)/[go.sum](go.sum).
No web framework, database server, embedded SQL database or plugin framework.
Go avoids requiring users to provision an interpreter; plain files allow an
externally synchronized personal directory without synchronizing a live database.

Keep cohesive responsibilities, initially as a few internal packages only where
useful: workspace resolution/configuration; saved resources/personal persistence;
entry planning and trust; platform launch; CLI presentation. Domain operations
return typed values/errors and do not print or prompt. CLI owns prompts and exit
codes. OS adapters know no ranking or saved-resource policy. Storage stays with
its owning feature; share only filesystem primitives actually used by both.
No mandatory folder template or separate package for every noun.

The entry flow is: resolve checkout -> validate configuration -> read resources
-> construct launch plan -> preflight and consent -> dispatch -> record result.
Inject launch, clock and storage boundaries for tests. Reuse the same resource
operations from a later browser adapter. Never give browser code its own database.

## Workspace and checkout identity

`workspace.id` is a newly generated UUID v4, normalized lowercase, stable across
clones/worktrees carrying the same configuration. Never derive it from a Git
remote, branch, name, or Git common directory. Same explicit ID means deliberate
shared personal resources; an independently forked project should get a new ID.
Names are display labels and need not be unique.

Resolve an existing path to an absolute, symlink-resolved directory. While walking
upward, take the nearest `workspace.toml`, stopping after inspecting the first
directory containing a `.git` file/directory or the filesystem root. A nested
configuration wins; a malformed nearest configuration is an error, not permission
to use an ancestor. Without configuration use that Git boundary, or the supplied
directory if no Git boundary exists. No Git executable or remote inspection is
required. Only regular config files are accepted; reject config symlinks in v1.

A checkout is a device-local UUID plus that canonical root path. Use filesystem
identity when comparing existing paths (including case aliases on insensitive
filesystems), not unconditional lowercasing. Symlink aliases resolve to one
checkout. Separate roots with the same workspace UUID remain separate checkouts.
Missing paths stay listed as unavailable and are never silently reassigned.

Without an explicit workspace ID, assign a directory workspace UUID in the local
registry. It is stable for that registered directory only; no cross-device linking
or move detection is promised. Personal records can use that UUID, but another
machine cannot discover the association from the path. `init` at that root reuses
the directory workspace UUID so existing saved pages keep their identity.
Otherwise `init` generates a UUID. It writes only `workspace.toml`, refuses to
overwrite any existing file, and does not stage/commit files or change ignores.
The initial recipe contains an editor resource at `.`. A retry after interrupted
initialization may finish its journaled registration, without rewriting the config.

If an already registered root changes explicit ID, fail with `identity_changed`
before launching or saving. Explain the old/new IDs. Explicit path opening with
`--accept-identity-change` rebinds that checkout after showing the change; it never
moves/deletes the old workspace's personal data or transfers trust. Interactive
confirmation may provide the same operation. Unknown moved roots register anew;
explicit IDs reconnect resources, while directory-only IDs require restoring the
old config/registry association manually. No silent identity migration.

## Configuration contracts

All configuration uses UTF-8 TOML with required integer `schema_version = 1`.
Reject unknown keys, unsupported versions, duplicate resource IDs, invalid types,
invalid UUIDs, and empty names. Missing configuration is distinct from malformed
configuration. Diagnostics name the file and field, without echoing full content.

Shared `workspace.toml` example (illustrative URLs, no executable commands):

```toml
schema_version = 1

[workspace]
id = "225116ad-90e9-42cd-9770-65d4b5d467ab"
name = "payments-api"

[[resource]]
id = "editor"
type = "editor"
path = "."

[[resource]]
id = "terminal"
type = "terminal"
path = "."

[[resource]]
id = "docs"
type = "url"
name = "API docs"
url = "https://example.com/docs"

[[resource]]
id = "chat"
type = "app"
profile = "team-chat"
```

`workspace` is required; `id` is optional for directory identity; `name` defaults
to root basename if omitted. `resource` defaults to an empty array. Resource IDs
are unique nonempty ASCII letters/digits/hyphens, names are optional display
labels. Each type accepts only its fields: URL requires `url`; editor/terminal
accept `path` defaulting to `.`; app requires `profile`. All shared resources
participate in entry. No resource status is needed in the shared recipe.
Paths must be relative, exist, and remain within the checkout after symlink
resolution; editor may target a file or directory, terminal requires a directory.
No globs, environment expansion, includes, shell expressions or repository argv.
Reject paths escaping the root, including through links. URL inputs everywhere
are absolute HTTP(S) URLs with a host and no userinfo or control characters.
Preserve query and fragment; normalize only scheme/host case and default port for
deduplication. Do not fetch titles or URLs in the background.

For a directory without configuration, synthesize one editor target at the root.
A present configuration with no resources intentionally launches only personal
pins. If the final plan is empty, report `nothing_to_open` without recording an
entry; inspection and saving still work.

Read-only inspection of an unregistered directory uses a transient identity,
clearly marked `registered: false`, with empty checkout/directory workspace IDs,
and never persists a generated UUID. An explicit config ID remains visible.
Initialization, saving or dispatched entry allocates the stable directory identity
under the registry lock. Failed dispatch of every resource does not register it.

Device configuration `config.toml`:

```toml
schema_version = 1
personal_data_dir = "~/Dropbox/Desky"

[launchers.editor]
executable = "/absolute/path/to/code"
args = ["--reuse-window", "{path}"]

[launchers.terminal]
executable = "/absolute/path/to/terminal"
args = ["--working-directory", "{path}"]

[apps.team-chat]
executable = "/absolute/path/to/chat"
args = []
```

These launcher values illustrate schema, not installed applications or universal
terminal flags. Launchers are optional; platform defaults are below. Executables
must be absolute local paths. `args` is a string array; `{path}` is allowed only
as an entire argument and required exactly once in editor/terminal profiles.
App profiles have literal arguments and no path substitution. App profile names
use the resource ID grammar. No shell command strings, environment overrides,
batch files, or shell interpreter evaluation in v1 profiles. A trusted installed
launcher script with an OS-supported shebang is permitted on Unix; it remains
user-controlled executable policy. Windows profiles require native executables.
Machine configuration is user-controlled executable policy: arguments can still
cause powerful application actions. It is not a sandbox.

The personal directory defaults below; a configured value must be absolute after
expanding a leading `~/` (or `~\` on Windows). No other variable expansion.
Relative paths and overlap with a selected checkout or device state are errors.
Changing this setting does not move, merge, or delete data: close Desky, copy the
entire personal directory, verify the copy, then change the setting. If a custom
directory is unavailable, fail rather than silently writing to the default.

Do not support `workspace.local.toml` initially. Machine profiles supply local
preferences without adding accidental repository execution or changing global
Git ignores. Effective precedence is built-in defaults then device configuration;
the shared recipe selects typed resources, never overrides executable profiles.
Personal pinned URLs are appended to the shared recipe; they do not rewrite it.

## Storage and durability

Locations use `desky` consistently, independent of the provisional CLI name:

| Platform | Configuration | Personal default | Device state |
|---|---|---|---|
| Linux | `$XDG_CONFIG_HOME/desky/config.toml` (fallback `~/.config`) | `$XDG_DATA_HOME/desky/personal` (fallback `~/.local/share`) | `$XDG_STATE_HOME/desky` (fallback `~/.local/state`) |
| macOS | `~/Library/Application Support/desky/config.toml` | `~/Library/Application Support/desky/personal` | `~/Library/Application Support/desky/device` |
| Windows | `%LOCALAPPDATA%\desky\config.toml` | `%LOCALAPPDATA%\desky\personal` | `%LOCALAPPDATA%\desky\device` |

Linux XDG base values must be absolute, per the
[XDG specification](https://specifications.freedesktop.org/basedir/0.8/).
Windows deliberately uses LocalAppData, not roaming storage. Configuration and
device state have no configurable relocation in v1. Only the personal directory
is intended for external filesystem synchronization. No credentials, OAuth,
transport, reconciliation service or application-managed synchronization.

Personal data is intentionally retained, including archived pages. Store each
resource as immutable JSON revisions at
`workspaces/<workspace-uuid>/resources/<resource-uuid>/<revision-uuid>.json`.
Use UUID v4 for resource and revision IDs; never names/URLs in filenames. Each
revision is a complete snapshot with this schema:

```json
{
  "schema_version": 1,
  "workspace_id": "225116ad-90e9-42cd-9770-65d4b5d467ab",
  "resource_id": "3165dd8d-b716-4eba-a2a9-2baecbb5c840",
  "revision_id": "42df2240-8fef-47a0-bbe6-a6d9ac97e98a",
  "parents": [],
  "type": "url",
  "url": "https://example.com/guide",
  "title": "Guide",
  "status": "saved",
  "created_at": "2026-10-02T01:00:00Z",
  "updated_at": "2026-10-02T01:00:00Z"
}
```

All fields are required except title (defaults to URL). Status is `saved`,
`pinned`, or `archived`. IDs must match the path. Times are UTC RFC3339, used for
display only, never conflict resolution. Initial revision has no parents; an
update cites the previous revision. Revisions and their predecessors are never
deleted or overwritten by v1. This small append-only representation avoids
multiple devices replacing one shared JSON database.

The current revision is the sole head (not referenced as a parent). Missing
parents mean incomplete delivery; multiple heads mean concurrent edits. Invalid
records, cycles, differing content with the same revision ID, and unknown schema
versions are errors. Preserve all files and show affected resource IDs; do not
pick a winner by clock. Readers inspect regular `.json` files even if an external
provider renamed them as conflict copies, deduplicating identical revision IDs.
The URL list includes conflicts. Entry skips each affected personal resource and
reports its ID/reason, while shared resources and healthy personal pins still
open. This explicitly replaces the original whole-entry conflict barrier at the
user's increment 3 request. Malformed, incomplete and unknown-schema resources
are similarly isolated. Failure to enumerate the overall store still blocks
entry because affected resources cannot be identified. Preserve every file.

`url pin/unpin/archive/restore <id> --resolve` explicitly selects that status for
a conflicted resource and writes a snapshot citing all current heads, provided
URL/title agree. Without the flag, conflicts fail. Differing URL/title, missing
parents, malformed files, or unknown versions require restoring valid files from
backup/provider history before mutation; never offer an automatic destructive
repair. No compaction or hard-delete command in v1. This does not implement a
sync protocol: providers must eventually deliver complete files, and provider
deletion, disk loss, malicious edits, or nonconverging delivery remain outside
Desky's guarantees. Users still need backups.

Device `registry.json` has `schema_version: 1` and `checkouts: []`. Each entry
contains `checkout_id`, `root_path`, `workspace_id`, `identity_source` (`explicit`
or `directory`), `name`, `last_entered_at` (nullable UTC time), and `entry_count`
(nonnegative integer). No browser data. Registration occurs on successful init,
URL save, or a dispatched entry, never as a side effect of list/info. Keep
directory identity allocation and registration in one locked transaction.
URL save persists registration before publishing personal data so interrupted or
failed publication cannot orphan a newly allocated directory identity. A failed
save can therefore leave registration, but never recency/count or approval.
On revision-write failure, remove only the resource directory created by that
attempt, using an atomic empty-directory removal that cannot unlink a file.
Never remove preexisting resource directories or any contents. A successfully
cleaned-up failure leaves no orphan conflict on retry; a revision published
before a late error remains available for normal URL deduplication. Failed cleanup
is reported. Process termination before cleanup can still leave an incomplete
directory; readers continue diagnosing it, including externally arriving empty
directories, rather than silently ignoring or sweeping them.
Initialization first publishes device `pending-init.json`: schema version 1,
the prepared checkout record, and base64-encoded generated config bytes. Under
the registry lock, init completes this journal before another registration, but
only if the config is absent or exactly matches those bytes. It then saves the
registry and removes the journal. A conflicting config is preserved and diagnosed;
there is no two-filesystem atomic transaction claim. Read-only commands never
recover or write pending state. See README for manual recovery procedures.
Trust records live separately in device `trust.json`: schema version and records
of checkout ID, approved launch-plan digest, and approval time. They do not sync.
Version 1 uses `records: [{checkout_id, digest, approved_at}]`, with a lowercase
UUID v4 checkout ID, hexadecimal SHA-256 digest, and UTC RFC3339 approval time.
Approval is persisted before dispatch; registration and recency follow successful
dispatch. If all launches fail before a new checkout is registered, its unused
approval record cannot approve a later newly allocated checkout ID.

For mutable device files, take an OS-released advisory lock, reread under lock,
write a same-directory temporary file, flush, then atomically replace using the
native platform operation. Keep one last known-good backup. Never reset malformed
state to empty silently; explain backup recovery. Lock contention has a bounded
two-second timeout. Use the same per-device lock for local personal mutations;
it does not coordinate other machines. Immutable personal writes use a temporary
file, flush and publish without replacement; ignore temporary suffixes when
reading. New-file publication currently uses hard links and requires filesystem
support; unsupported filesystems fail explicitly. Flush directory metadata where
supported. Report write/flush failures
without claiming success. Readers tolerate only fully published files.

Default to user-only permissions where supported; inherit the user's private
application directory ACL on Windows. No file encryption is promised. URLs may
contain sensitive query strings; store only intentionally supplied data, never
log full URLs by default. Do not store checkout paths, timestamps of browsing,
trust or launcher executables in personal records. No saved data expires.

## CLI, resolution and output

First-release command contract (README and CLI help distinguish implemented commands):

| Command | Contract |
|---|---|
| `desk` | Interactive recent picker; no current-directory shortcut |
| `desk <selector>` / `desk open <selector>` | Same resolver and entry behavior |
| `desk open --workspace <uuid> [--checkout <path>]` | Exact identity for scripts; checkout must belong to it |
| `desk init [--name <name>]` | Initialize current directory explicitly; no parent search for writing |
| `desk list [--json]` | Known checkouts with workspace IDs, paths, availability and recency |
| `desk info [<selector>] [--json]` | Current checkout by default; identity, config/data paths, effective resources and trust state |
| `desk url add <url> [--title <text>] [--pin] [--workspace <uuid>]` | Save intentionally; infer current checkout unless exact workspace supplied |
| `desk url list [--all] [--json] [--workspace <uuid>]` | Saved/pinned by default; `--all` includes archived; always surface conflicts |
| `desk url pin/unpin/archive/restore <id> [--resolve] [--workspace <uuid>]` | Set pinned/saved/archived/saved respectively; keep resource ID/history |
| `desk config path` | Print device config path, even before it exists |
| `desk config get <key>` | Read effective `personal_data_dir`, `launchers`, or `apps`; edit TOML directly to change them |

`--help` and `--version` exist. Entry additionally supports `--dry-run`, `--json`,
`--trust <digest>`, and `--accept-identity-change`. Dry run validates and presents
the plan/digest but performs no registration, approval or launching. Its proposed
directory-only identity is marked ephemeral until registration. `--workspace`
may address an explicit UUID with personal data but no local checkout for URL
operations; opening always requires a local checkout. Unknown UUIDs fail.
Exact UUID URL operations address personal data only, without selecting an
arbitrary checkout's shared recipe. Current-directory URL listings also include
that checkout's shared URLs.

Reserved commands take precedence at the root; use `desk open config` for a
workspace named `config`, `desk ./config` for a path. `--` ends option parsing.
Within a selector, an existing directory takes priority over names; an existing
file is an invalid target. Explicit path syntax (absolute, `.`, `..`, separators,
leading `~`) that does not exist fails as a path rather than fuzzy matching.
Expand only a leading home marker. For remaining strings: exact workspace UUID,
then exact case-insensitive name, then case-insensitive substring/subsequence
matches over name and known path. Names with multiple matches are ambiguous.

Interactive ambiguity always presents choices including paths; fuzzy matches
always require selection, even when only one remains. Scripts accept only exact,
unambiguous identities/names/paths and fail rather than guessing. When a workspace
has several available checkouts, use the containing checkout if invoked inside
one, otherwise prompt interactively or require `--checkout` for scripts. Never
choose a branch based on remote URL or silently choose an arbitrary checkout.

The picker lists available checkouts with consecutive identical workspace names
grouped under a heading and distinct numbered paths selectable. Grouping never
reorders MRU results. Sort by most recent successful entry, then entry
count, then name and canonical path for stable ties; unentered checkouts sort
last. This is MRU-first, a deliberate simpler alternative to an unproven frecency
formula. Begin with a line-oriented numbered selector supporting typed query
filtering, Enter for the top displayed item, and explicit cancellation. Preserve
accessibility and speed without requiring an external fuzzy finder or shell hook.
Empty registry explains `desk .`/`desk init`; bare command without a TTY fails with
guidance to `list` or explicit `open`. Never read piped input as implicit consent.

URL add defaults to saved, `--pin` to pinned. Adding an existing normalized
personal URL returns its ID without changing status/title; use a status command
for changes.
Deduplicate concurrent identical additions for launching, but retain/list their
distinct resource IDs. Shared URLs are read-only to URL mutation commands; display
origin and direct edits to workspace.toml. Personal status changes never mutate
the shared recipe. Open shared URLs plus personal pins once per normalized URL;
saved and archived personal pages do not open. No title scraping or capture.
Human URL output uses an untitled placeholder when the title defaults to the URL;
JSON inspection contains full intentional snapshots. `:q` cancels the picker and
`/` clears its filter. JSON and redirected input never enable selection or consent.

Structured results have `schema_version: 1`, `ok`, and either `data` or `error`
(`code`, `message`, optional `details`). JSON mode disables prompts and produces
one JSON document on stdout, including failures; human diagnostics go to stderr.
Entry data includes each resource's requested/dispatched/failed result and any
state-write warning. List/info/URL list use the same identity and resource shapes,
including origin and availability, without exposing executable secrets. Escape
control characters in all human-visible names/titles/paths; never emit raw terminal
escape sequences from configuration. Normal human success output stays concise.

| Exit | Error category |
|---|---|
| 0 | Requested operation completed (dispatch acknowledged for entry) |
| 2 | Usage or interaction required, including bare command without TTY |
| 3 | Workspace/checkout not found or unavailable |
| 4 | Ambiguous workspace or checkout |
| 5 | Invalid configuration, identity changed, or nothing to open |
| 6 | Repository trust required/mismatched digest |
| 7 | Unsupported platform action or missing launcher |
| 8 | Launch dispatch failure, including partial launch |
| 9 | Resource/state IO, conflict, schema or lock failure |
| 130 | User cancellation; no launch or mutation |

## Launch policy and platform boundary

Opening repository content can activate editor extensions, terminal startup files,
network requests, and application-specific behavior even without shell commands.
Do not claim typed configuration makes an untrusted repository safe. Before first
entry, show checkout, resource types/targets, resolved launcher profiles and their
arguments. Require affirmative interactive approval or `--trust <digest>` using
the exact digest from a previously inspected dry run. Bind trust to checkout ID
and SHA-256 of canonical effective shared launch plan plus device launch profiles
and canonical root. Recompute after preflight and dispatch only that immutable
plan; changes require new consent. No blanket `--yes` or repository-written trust.
Changing source formatting alone need not invalidate trust. Personal URL pins are
explicit user data and do not require repository reapproval, but remain visible
in the plan; external personal-directory writers can therefore influence entry.

Preflight validates paths, profiles and trust, and classifies personal resources
before dispatching anything. Conflicting/uninterpretable personal resources are
reported as skipped; all other preflight failures still dispatch nothing.
Shared resources run in listed order; healthy personal pins
follow in resource-ID order. At runtime, attempt remaining independent items after
a launch failure, report each result, and return a failure code. Do not undo,
close, or supervise applications. Record recency/count once when at least one
dispatch succeeds, including partial success. If state persistence then fails,
report the successful launches and state failure; never suggest a blind retry.
Preflight errors cause no launch; dry run causes no writes. For mixed runtime
failures, launch failure (8) takes precedence over state-write failure (9), while
structured output contains both.
An entry with skipped personal resources returns resource failure (9) after
attempting the unaffected plan; dispatch failure (8) takes precedence. Dry run
shows the skipped IDs/reasons without dispatch or writes. If only skipped
resources remain, return resource failure without recording entry. Personal pins
are re-read after consent and excluded from the shared recipe approval digest.

The platform boundary accepts only `OpenURL(url)`, `OpenEditor(path, profile)`,
`OpenTerminal(directory, profile)`, and `OpenApp(profile)` and returns dispatch
results. Use native argument arrays and native Windows quoting; no `sh -c`,
`cmd /c`, PowerShell evaluation or string concatenation. Go's
[os/exec](https://pkg.go.dev/os/exec) does not implicitly invoke a shell; preserve
that property. Never resolve executables relative to the checkout. Built-in
launcher lookup uses trusted system locations or absolute PATH entries outside
the checkout, rejecting current-directory lookup. Explicit profiles are local
user policy and take precedence. Child working directory is the target directory
for editor/terminal, checkout root for app, and neutral home for URL helpers.

| Action | macOS | Windows | Linux desktop |
|---|---|---|---|
| URL | `/usr/bin/open` with one validated URL | `ShellExecuteW` open verb | `xdg-open` with one validated URL |
| Editor default | `code --reuse-window <path>` if installed | Native Code executable if available; do not execute `code.cmd` | `code --reuse-window <path>` if installed |
| Terminal default | `/usr/bin/open -a Terminal <directory>` | `wt.exe new-tab -d .` with child CWD set to the directory | Explicit terminal profile required; terminal flags vary |
| App | Explicit native executable profile | Explicit native executable profile | Explicit native executable profile |

If the default editor or terminal is absent, explain how to configure a profile;
never install it automatically. Native app bundle launching can be configured
through macOS `/usr/bin/open` with literal app arguments. Default profiles must
be exercised on native hosts before claiming support. macOS Terminal's directory
behavior is an implementation smoke-test gate, not a claim verified here.
Windows URL launching uses the documented
[ShellExecuteW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecutew)
boundary; terminal directory arguments follow
[Windows Terminal documentation](https://learn.microsoft.com/en-us/windows/terminal/command-line-arguments).
The Windows default passes `.` because Terminal treats semicolons in arguments as
command separators. The full checkout path travels only through the native child
working directory. Explicit user profiles remain application-specific executable
policy and may require their own handling of application-level argument syntax.
Linux delegates URL preference to
[xdg-open](https://wiki.freedesktop.org/www/Software/xdg-utils/).

Helpers that return promptly are awaited for dispatch errors. Direct application
processes are detached with standard handles disconnected and handles released;
success means OS process creation, not later GUI readiness. Each platform must
verify its detachment semantics. Bound dispatch-helper waits (five seconds),
report timeout as outcome unknown, and do not automatically retry or terminate a
possibly opened application. Reuse/focus follows application behavior; duplicate
terminal windows or tabs are possible. No PID/window tracking and no `isOpen`.
The parent shell's directory never changes. WSL/SSH/headless sessions and Windows
UNC/network checkouts are outside initial launch guarantees; unsupported actions
fail clearly, while inspection and URL storage should still work.

## Browser seam and deferred decisions

Expose internal operations `ListWorkspaces`, `ListResources(workspaceID)`,
`SaveURL(workspaceID, url, title, pin)` and `SetURLStatus(workspaceID, resourceID,
status, resolveConflict)`. CLI uses these operations now; a future Firefox native
messaging adapter can call them without spawning a CLI per request. Require exact
workspace IDs in integration requests and explicit selection of unknown targets.
Transport schema/versioning, host installation and extension allowlisting belong
to that increment. [Mozilla native messaging](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/Native_messaging)
is a candidate transport; no localhost listener, daemon or installed native host
is part of this release. The adapter cannot alter launch trust or run commands.

Browser automatic capture remains off by default and absent here. A later design
must exclude private/incognito capture, show active capture, support pause, and
keep observed sessions device-local, separate from intentional resources. Clearing
observed state must never clear saved/pinned/archived data. Browser-vendor sync is
the only contemplated mechanism for browser-specific synchronized state.

No material product decision blocks the planned increments. Revisit executable
branding and minimum OS/CPU support before public distribution; validate native
launch behavior before claiming a supported platform. Decide permissions and
consent for capture before implementing it, and action execution/trust before
adding `run`. Alias semantics, previous-workspace behavior, richer picker UX,
clipboard privacy, local override merge rules and revision compaction can wait
for demonstrated need. Do not let these deferred choices grow v1 silently.
