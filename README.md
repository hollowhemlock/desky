# Desky

A cross-platform project-context launcher for fast re-entry into a working
directory and its editor, terminal, applications, and saved web resources.

Current state: increments 1-3 and increment 4's native adapters are implemented.
The Go CLI selects recent projects, retains personal URLs, and requests editor,
terminal, named application and URL launches after recipe approval. Windows has
desktop smoke evidence; macOS/Linux desktop qualification remains open. The
exact verification matrix and outstanding release gates are in [PILOT.md](PILOT.md).

## Start here

| Document | Role |
|---|---|
| [AGENTS.md](AGENTS.md) | Required reading order, maintenance rules, and the project-local autonomy pilot policy |
| [plan.md](plan.md) | Product intent, boundaries, identity, storage scopes, privacy, and requested next deliverable |
| [namespace.md](namespace.md) | Proposed terminal interface, command semantics, and first-version versus future scope |
| [SPEC.md](SPEC.md) | Selected release boundary, architecture, identity, schemas, CLI, trust, platform launch and browser seam |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | Incremental acceptance, planned verification, Git integration context and readiness review |
| [internal/README.md](internal/README.md) | Implemented capabilities, ownership, code entry points and tests |
| [PILOT.md](PILOT.md) | Ready-to-use first-task prompt and evaluation sequence |

The source documents mix firm constraints with examples and possible future
features. Preserve their distinctions when deriving the specification. Update
this entry point as concrete specifications and implementation become available;
point to their authoritative sources instead of maintaining parallel inventories.

## Build and use

Install the Go version specified by [go.mod](go.mod) or a compatible newer release.
From this repository, run `go run ./tools/verify`. The
[verification runner](tools/verify/main.go) checks formatting, vets and tests the
module, then builds `bin/desk.exe` on Windows or `bin/desk` on macOS/Linux. It is
also used by [CI](.github/workflows/verify.yml). With a supported C toolchain,
`go run ./tools/verify -race` additionally runs the race detector. Add `-cross`
to compile all three OS targets for amd64 and arm64; this checks compilation,
not native launch behavior. Dependencies
are pinned in go.mod/go.sum; the first run may need network access to download them.

Run the resulting executable with `--help` for the implemented command catalog,
owned by [the CLI](internal/cli/cli.go). For example, run `desk init` in a project
directory, then `desk info --json` or `desk list`. Here `desk` means the built
executable's absolute path, or a binary you have placed on your PATH; the build
does not install it or change shell settings. Init creates a workspace.toml with
a stable UUID and editor resource, and registers the checkout outside the repo.
It never stages that file in Git. Repeating a completed init refuses to overwrite.

Use `desk config path` to locate device TOML, and `desk config get personal_data_dir`
to inspect the effective personal directory. Edit configuration using the schema
in SPEC.md. Metadata inspection does not create state or register a checkout;
unregistered directory identities have empty IDs with `registered: false`.
Successful entry registers a checkout and updates recency once, including partial
launch success. Bare `desk` opens a numbered recent picker; type a query to filter,
Enter for the first displayed item, `/` to clear, or `:q` to cancel. Ambiguous and
fuzzy selectors use the picker in a terminal; scripts require exact selectors.
Info reports `approved`, `approval_required`,
or `not_evaluated` with a reason when launch preflight cannot complete.

On Windows, inspect `desk open . --dry-run` first. Run `desk .` in an interactive
console to approve the displayed recipe, or pass the dry run's digest with
`desk open . --trust <digest>`. JSON mode and redirected input never prompt.
Approval is local to that checkout and effective recipe; changing a target or
launcher profile requires new approval. Neither dry run nor info writes state.
The default editor is native VS Code; the default terminal is Windows Terminal.
Configure absolute native `.exe` paths in device TOML when defaults are unavailable.
Shared HTTP(S) URLs use the default browser. See SPEC for profile schemas.

The macOS/Linux adapters use `code --reuse-window` from an absolute PATH entry
outside the checkout by default. macOS uses `/usr/bin/open` for URLs and Terminal;
Linux uses `xdg-open` for URLs and requires a configured `launchers.terminal`
profile with the terminal's own directory flag and one `{path}` argument. Unix
profiles may use executable shebang scripts. Helpers have a five-second dispatch
deadline; timeout means the outcome is unknown and never kills the application.
The default Unix `code` launcher is also awaited as a dispatch helper. Use native
launching only in a local desktop session. Preflight rejects SSH, WSL and detected
headless sessions; metadata and URL saving still work there. Windows requires a
visible process window station.
Display/session detection does not prove that a GUI service or application is
healthy. macOS Terminal directory behavior and Linux desktop behavior still need
the native smoke checks described in IMPLEMENTATION.md before support is claimed.

Entry reports dispatch, not GUI readiness or process ownership. It attempts
remaining resources after a runtime failure and reports all results. A state-write
failure after launch is reported separately; inspect the results before retrying
because applications may already be open. Windows network checkouts are unsupported.
Conflicting or damaged personal URL resources are reported as skipped, while
shared resources and healthy personal pins still open. Such entry returns exit 9
(or 8 if dispatch also fails); JSON includes every skipped/dispatched/failed result.

Save an intentional URL with `desk url add <url> --title <text>`; add `--pin` to
open it on entry. `desk url list --all` includes archived URLs and always shows
conflicts. `pin`, `unpin`, `archive`, and `restore` take a personal resource UUID;
unpin/restore set it to saved. Saved and archived URLs do not open. Existing
normalized personal URLs keep their ID, title and status when added again.
Shared URLs stay read-only; entry deduplicates them with personal pins. No URL
fetch, title scraping or clipboard access occurs. Use `--json` to retrieve full
URL snapshots; ordinary output omits URLs used as default titles.

Personal revisions live outside the checkout in `personal_data_dir`. A configured
custom directory must already exist; unavailable storage never falls back to the
default. To relocate it, close Desky, copy the whole directory, verify the copy,
then edit device TOML. An explicit workspace UUID reconnects copied resources on
another device; directory-only identity requires its original local association.
`url --workspace <uuid>` addresses personal data without choosing a local checkout;
omit it to include the current checkout's shared URLs in listings.

For concurrent status edits, inspect `url list --all --json`, then explicitly use
`url pin|unpin|archive|restore <id> --resolve` when all heads agree on URL/title.
Resolution appends a revision citing every head. All original files remain.
Malformed records, missing predecessors, differing URL/title or unknown schemas
require recovery from backup/provider history; Desky never picks a timestamp
winner or automatically deletes/rewrites conflicting data. Failure to enumerate
the overall store still blocks entry because affected resources cannot be known.

## Recovery and verification limits

Init uses a device-local `pending-init.json` journal across config/registry writes.
If interrupted, rerun init: it completes only the exact prepared configuration,
preserving its original name and IDs. It never overwrites a differing config.
Until recovery completes, read-only commands expose only committed registry data.
If pending config and actual config differ, preserve both and restore the prepared
config (the journal's base64 `config` field) before retrying; do not discard personal
files. An unavailable pending checkout must be restored before another init.

Registry corruption fails explicitly. Preserve the damaged file, inspect
`registry.json.bak` in the device state directory, and restore a known-good backup
while no Desky commands are running. State locations and durability limits are
defined in SPEC.md. Initial filesystem support requires hard links for publishing
new config/journal files; unsupported filesystems fail without overwriting data.
Approval corruption similarly preserves `trust.json`; inspect its `trust.json.bak`
before restoring a known-good copy. Restoration never transfers another checkout's
approval. An unfinished init must be recovered before entry can mutate device state.
Saving in an unregistered directory persists identity before publishing its first
URL. A failed URL write can leave that registration, but no entry count or trust;
retrying retains the same identity. Clearing device state never deletes personal
revisions; directory-only workspaces need their original association restored.

If revision publication fails, Desky removes the resource directory only when
that attempt created it and it is still empty. Retry then saves without an orphan
conflict. Existing directories and all files, including external deliveries and
revisions published before a late write error, are preserved. A retry reuses an
already-published matching URL. If cleanup fails or the process stops before it
runs, the remaining directory is still diagnosed as incomplete; preserve it and
inspect backup/provider history rather than deleting potentially incoming data.

The verification suite covers concurrency/recovery, trust mutation, partial
launches, state-write failures, real argument/CWD preservation, detached-child
survival and dispatch-helper timeouts. Native CI runs it on Windows, macOS and
Linux, including Unix lookup/session/URL-helper tests and private file permissions.
Helper tests do not establish native desktop launch support. The desktop smoke
evidence, exact app versions and remaining increment 4 qualification work are in
[PILOT.md](PILOT.md). Increment 3 conflict, picker and performance evidence is also
recorded there; measured latency is not a performance guarantee.
