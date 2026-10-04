# Implementation plan

Status: increments 1-3 are implemented. Increment 4 adapters and automated checks
are implemented; native desktop qualification remains **open** (matrix in PILOT.md).
[SPEC.md](SPEC.md) is authoritative for behavior and technical decisions.
Do not scaffold the entire plan at once. Each increment should leave a usable,
verified slice and update README navigation to its actual code and checks.

## 1. Inspect and register workspace identity

Establish the Go module, pinned TOML dependency, executable entry and focused
workspace/configuration ownership. Implement `init`, `info`, `list`, configuration
path/get, directory resolution, local registry and typed/JSON errors. Establish
atomic persistence and the smallest test seams needed for later personal writes.
No real launch adapters or URL management yet. Commands not implemented must
return explicit unsupported/usage errors, never placeholder success.

Acceptance:

- In a disposable directory, init creates only workspace.toml and external local
  registration; repeating it refuses without changing either file. Reopen info
  in a fresh process and observe the same workspace and checkout IDs.
- Two worktrees with the same explicit UUID share workspace identity but keep
  distinct checkout paths. Two unconfigured directories and two repositories
  with identical remote text do not share identity. Symlink/case aliases resolve
  to the same existing directory where the filesystem supports them.
- A nested config wins; nearest malformed config and changed ID fail explicitly.
  Info/list do not mutate state. Reserved names and explicit missing paths behave
  as specified. Non-TTY ambiguity never chooses a checkout.
- Inject interrupted writes, malformed state, permission failure and concurrent
  registration. Preserve readable old state/backup and never invent success or
  erase existing IDs. Verify no state is written inside the checkout.

Verification: table-driven unit tests for resolution/schema/errors, filesystem
integration tests with isolated home/config/state roots, and CLI subprocess tests
for persistence, output and exit status. Use an internal location resolver to
inject roots; tests must never use real personal data. Update README with actual
runtime prerequisites and the authoritative verification entry point once it
exists. Completion makes workspace metadata inspection usable, not entry.

## 2. Open a directory safely on the current platform

Implement pure launch planning/preflight, digest approval, dry run and dispatch
results. Add the current host adapter, configured editor/terminal/app profiles
and URL launch. Implement positional/exact `open` and MRU bookkeeping. Native
adapters for other OSes are increment 4, not stub implementations returning success.

Acceptance:

- A path with spaces, Unicode, shell metacharacters and leading punctuation
  reaches the requested executable as one argument; injection fixtures create
  no marker files. Reject escaping paths, unsafe URL schemes and repository argv.
- Unapproved recipe opens nothing; inspection/dry run opens nothing and writes
  nothing. Approval matches exactly one checkout and effective recipe. Changing
  a target or local profile invalidates approval. Another worktree needs approval.
- Open the chosen editor/terminal at the selected root and a URL in a real desktop
  session. Repeat entry without creating process ownership or teardown state.
  The invoking shell's directory is unchanged.
- Preflight failures dispatch nothing. A simulated later launch failure leaves
  successful items reported, attempts independent remaining items, and records
  entry once. Failed state writes report that apps already launched.

Verification: fake-launcher unit/contract tests, helper-executable argument/CWD
integration tests, trust mutation tests and a native manual smoke check. The smoke
record names OS, app versions and observed outcomes. A helper launch is not proof
that a GUI app focused correctly. Once this passes, explicit directory entry is
the first usable launcher increment; other platforms remain unverified.

Completed on Windows/amd64. The local verification runner passes; launch tests
cover dry-run/cancellation, digest changes, checkout-specific approval, changed
identity, preflight rejection, partial dispatch, recency write failures and
concurrent entries. Native subprocess tests exercise Windows argument quoting,
child CWD, disconnected handles and survival after parent exit. The desktop smoke
and its user-observed result are recorded in PILOT.md. Native adapters on other
platforms explicitly fail with exit 7. Personal workspace resources are now
interpreted by increment 3 below.

## 3. Select recent projects and retain personal URLs

Implement the MRU-first picker, filtered ambiguity selection, and URL add/list/
status operations through the resource domain seam. Add immutable revision
persistence, personal-directory configuration, conflict handling and merging
shared URLs with personal pins for entry. No browser transport or clipboard.

Acceptance:

- Register multiple projects, enter them in a known order, and select by number
  or filtered query. Equal times have deterministic ties. Duplicate names show
  checkout paths; redirected stdin never triggers selection or consent.
- Save a URL, restart the process, and retrieve its ID/title/status. A pin opens;
  a saved or archived page does not. Unpin/restore preserve history. Shared data
  stays read-only and duplicate URLs open once. No network fetch occurs on save.
- Copy personal files into another isolated machine fixture: an explicit workspace
  UUID reconnects URLs despite a different checkout path; registry/MRU/trust are
  absent from the copied files. Directory-only identity does not merge by name.
- Simulate two devices editing one resource independently. Union the revision
  files in different arrival orders: both versions survive, conflict is visible,
  and entry skips affected URLs while opening unaffected resources. This replaces
  the original whole-entry barrier per the increment 3 request. Explicit resolution
  creates a head referencing both; equal final files yield equal state regardless
  of timestamps.
- Incomplete delivery, renamed conflict copies, unknown schema, disk-write errors
  and unavailable custom storage never silently reset or discard personal data.
  Clearing/recreating device state does not delete saved resources.

Verification: resource-state and URL validation unit tests; isolated multi-root
filesystem/subprocess tests for restart, concurrent saves and failure injection;
CLI tests for JSON/errors; interactive smoke check for picker usability. Measure
warm list and dry-run latency with 1,000 checkout entries and 1,000 current URL
resources on a named machine; aim for under 100 ms for local computation excluding
process startup, OS dispatch and slow externally synchronized storage. Report
measurements, not an untested hard performance guarantee. If UX is slow, optimize
measured work before adding a persistent service.

Implemented in the resource domain, CLI picker and launch integration, routed
through internal/README.md. Tests cover process restarts/concurrent saves, status
history, replica union/resolution, invalid/incomplete records and provider conflict
copies, custom storage, shared/personal deduplication, trust stability, partial
entry and noninteractive selection/consent. Windows terminal smoke and measured
1,000-checkout/1,000-URL latency are recorded in PILOT.md. Native launch adapters
on other platforms remain increment 4; local race testing lacks enabled CGO.

## 4. Complete and qualify the platform boundary

Implement remaining native adapters and persistence/locking details, exercising
the same contract tests. Use current supported desktop hosts for macOS, Windows
and Linux; document the exact OS/architecture/app matrix actually exercised before
stating support. Linux terminal launch requires a configured profile. No installers,
signing, publishing or production deployment are implied by this increment.

Acceptance:

- On each native host, request an editor, terminal at a directory with spaces,
  HTTP(S) URL and named desktop app; verify visible behavior, prompt completion,
  repeat-open behavior and control returning to the CLI. Confirm no spawned app
  is killed when the CLI exits and no attached standard handles keep it alive.
- Test missing launcher, absent desktop session, native URL failure, dispatch
  timeout, and platform-specific argument quoting/detachment. Unsupported actions
  return the documented error instead of silently falling back to a shell.
- Exercise filesystem publication/locking and recovery on each OS. Concurrent
  commands must not lose device registry records. Corrupt personal input remains
  intact and diagnosed. Re-run the privacy/storage-boundary checks on each host.

Verification: native Go test suites plus recorded desktop smoke results. Build
each target to catch compile drift, but never call cross-compilation a launch test.
If a native host is unavailable, mark that platform unverified and leave its release
gate open. Automate host-independent checks in CI when implementation begins;
desktop interaction may remain an explicit release check. Public release also
requires a separate branding/support/distribution decision and authorization.

Implementation: macOS/Linux adapters resolve native profiles and trusted default
launchers, reject detected unsupported sessions, dispatch URL helpers with a
bounded wait, and detach direct children without shell evaluation or teardown.
Linux requires an explicit terminal profile. Shared native process tests cover
argument/CWD preservation, parent exit, disconnected output handles, helper failure
and timeout survival. Unix tests cover safe lookup, shebang execution, URL failure,
session rejection and private file permissions. Existing native flock/rename/fsync
and hard-link publication remain the persistence implementation; recovery,
concurrent registry/resource and privacy tests run on each CI host.

The runner's `-cross` option builds Windows/macOS/Linux for amd64 and arm64;
Linux CI also runs the race detector. These are executable verification gates,
not a desktop support claim. See PILOT.md for actual check results and the exact
desktop matrix. Complete the remaining native desktop smoke checks before marking
this increment fully qualified; no installer or release publication is included.

Optional [VM development tooling](tools/vm/README.md) prepares an Ubuntu Desktop
guest from Windows, transfers committed source and records isolated qualification
runs. This is separate from Desky's application boundary. Deterministic helper
checks do not complete native desktop acceptance; real installation, provisioning
and observed qualification remain explicit gates.

## Verification and completion discipline

Start with affected Go package tests; the full gate is now maintained by
[tools/verify/main.go](tools/verify/main.go), invoked as documented in README.
Run race checks on hosts/toolchains that support them for changed concurrent
persistence. CI invokes the same runner on the three target OSes and adds a
Linux race check; a configured workflow is not evidence of a successful run.
An interface test must exercise observable behavior, not just mirror private code.

Each increment documents actual implementation, constraints and verification via
README routes; keep later increments visibly planned. Feature-specific ownership
should contain its code, tests and contracts when the implementation grows, while
cross-feature launch integration remains discoverable. Moving an internal file
should not require rewriting root guidance; changing verification orchestration
should not require editing copied command lists. No speculative directory tree.

Use short-lived task branches from the current adopted integration baseline.
For this pilot the baseline is `autonomy-pilot` at `cc5e0ae`; the specification
branch is `docs/initial-specification`. Subsequent work should include this
specification commit or its merged successor, rather than assume it already
exists on main. Remote default is currently `origin/main`; no merge or PR is
authorized by this plan. Do not introduce permanent integration/release branches
as part of specification work. Follow AGENTS for commits/pushes and preserve
unrelated work. Record each pilot task in [PILOT.md](PILOT.md).

## Specification review and readiness

Review type: staged implementation plan for a new local CLI. Priorities are
identity/data durability, trust before effects, deterministic scripting, useful
increments and evidence for native platform claims. Source requirements are
evidenced by the two preserved briefs; platform behavior remains only partially
evidenced by vendor documentation until tested.

The review resolved these concrete risks in the specification: bare-command
ambiguity; directory-to-explicit identity preservation; multiple checkout
selection; shell/repository execution exposure; external concurrent writes;
partial launch and recency errors; missing terminal defaults; and silent
cross-platform success claims. Immutable personal revisions cost additional files
but avoid overwriting concurrent intentional data; no general sync service is
introduced. Repository actions and browser capture stay outside the release.

**Initial specification review: Ready to begin increment 1.** The sequence has bounded
acceptance and explicit native verification gates. No unanswered material product
choice blocks implementation; successful platform behavior and public release
readiness are not established by this review.

**Next action:** finish increment 4 desktop qualification on native macOS/Linux
hosts and the remaining Windows desktop cases listed in PILOT.md. Preserve
identity, conflict isolation and consent contracts while recording observed
application behavior and exact OS/app versions. The performance aim remains a measured
target, not a release claim; no persistent service has been introduced.
