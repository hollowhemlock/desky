# Desky autonomy pilot

Status: the first architecture/specification task is documented in
[SPEC.md](SPEC.md) and [IMPLEMENTATION.md](IMPLEMENTATION.md). Workspace metadata
and Windows entry, personal URL persistence and selection (increments 1-3) are
implemented. Increment 4 adapters are implemented with native desktop qualification
still open. Browser integration remains planned. Evidence appears below.

Baseline: `97f7edb443fde605932fc08378e6021b403d50cd` contains the original README,
product brief, and terminal namespace design. The current policy is the v0.2
snapshot in [AGENTS.md](AGENTS.md); the specification and increments 1-3 used v0.1.
This trial uses the existing Codex tools and permissions;
no global policy, custom agent roles, or additional skill installation is needed.

## First task: architecture and specification

Open a fresh Codex task rooted in this repository and send the following prompt.
The repository documents supply the product brief; no retelling of the autonomy
design conversation is required.

> Read AGENTS.md, README.md, plan.md, and namespace.md before designing Desky.
>
> Carry out the architecture/specification task requested at the end of plan.md.
> Write a concrete, proportionate specification and an incremental implementation
> plan into this repository. Keep this task documentation-only; do not scaffold
> application code or install its toolchain yet.
>
> Choose a suitable initial stack, architecture, storage mechanisms, and planned
> verification approach. I delegate these technical decisions and waive routine
> proposal approval, including in any bootstrap workflow you use. Ask only about
> unresolved material product or operational consequences. Resolve minor
> reversible details conventionally and explain noteworthy tradeoffs.
>
> Follow the source documents' product boundaries and distinguish requirements
> from provisional names, examples, and optional future features. Define the
> smallest useful first release instead of adopting the whole eventual command
> tree. Keep responsibilities cohesive and project facts authoritative in one
> place; do not create speculative infrastructure or a mandatory folder template.
>
> Cover the configuration and state schemas and locations; durable workspace
> versus checkout identity; the CLI surface and resolution/error behavior; the
> minimal macOS, Windows, and Linux launch boundary; and a narrow seam for later
> browser integration. Explain relevant data durability and repository-config
> execution/trust consequences without adding hosted services or browser capture
> to the initial scope. Preserve the original brief and namespace document.
>
> Define observable acceptance and a suitable verification approach for each
> implementation increment. Update README.md and navigation to the resulting
> documents. Review the specification against both source documents and resolve
> routine technical gaps yourself. Clearly distinguish open product decisions,
> chosen technical defaults, and behavior that remains unimplemented.
>
> Commit the task-related documentation on a task branch and normally push it to
> the configured remote. Report actual validation, unresolved material decisions,
> and the commit/push result. Do not claim application tests or cross-platform
> behavior were verified during this specification task.

This task tests requirements interpretation and autonomous technical planning.
It does not demonstrate implementation correctness. The next ordinary task can
request the first implementation increment; no repeated bootstrap prompt is
needed.

## Continue the trial

After the specification task, use normal product requests to:

1. Implement the first usable increment and meaningful executable verification.
2. Add another capability and inspect ownership and change locality.
3. Make a change spanning boundaries or shared configuration; check for drift.
4. Ask a fresh agent, without previous conversation history, to extend the app
   using only maintained repository context.

Keep model/settings, tools, and permissions comparable and record deliberate
changes. Use a few acceptance examples for each task without specifying internal
implementation. For a promising policy, compare representative tasks with the
previous policy on separate copies of the same starting state.

## Evidence record

Create one brief entry per task with:

- Task link, policy revision, model/settings, and starting/ending commit.
- Acceptance result backed by reviewed behavior, diffs, and actual check output.
- Unnecessary questions, missed material questions, and instruction conflicts;
  count tool approval prompts separately.
- Ownership or drift problems, and whether new tooling or context helped later
  work. Record concrete paths rather than judging a folder name alone.
- Human answering/review/rework effort and available runtime/cost information.

Preserve unrelated work and use disposable fixtures for deliberate failure cases.
An unauthorized consequential action, lost user work, or false verification
claim fails the candidate for adoption until addressed and retested. Otherwise
judge correct completion, appropriate questions, bounded improvements, and easier
continuation without chat history. A second materially different repository is
needed before treating this pilot as evidence for global adoption.

### 2026-10-01: initial architecture/specification

- Task: current Codex task invoked via this file; a shareable task link was not
  exposed or created. Policy v0.1 unchanged. Runtime identifies the GPT-6 family;
  exact model variant, reasoning setting, token/cost totals and independently
  measured human effort are unavailable.
- Start: `cc5e0ae28c1bb3b6dd976ca61165d7ea6fd6d492`, clean `autonomy-pilot` branch.
  End: the documentation commit containing this entry on
  `docs/initial-specification` (resolve with `git log -1 --format=%H -- PILOT.md`).
  The task's final report records the resulting commit hash and push outcome;
  this entry does not predict remote success.
- Delivered: selected defaults/contracts in SPEC.md; four increments with
  observable acceptance and readiness review in IMPLEMENTATION.md; README routes
  and project-specific AGENTS reading guidance. Original plan.md and namespace.md
  are unchanged. No toolchain, application code, manifest or CI was added.
- Actual checks: local Markdown links resolved; all three SPEC TOML/JSON examples
  parsed using Python 3.13 standard-library parsers; new-document code fences and
  whitespace checked; source-document diff empty. Staged whitespace/diff review
  is the final pre-commit gate. These are document checks, not application tests.
- Source coverage review: product identity/storage/privacy and non-goals map to
  SPEC identity, configuration, storage and browser sections; namespace selection,
  script output and errors map to its CLI section; launch boundaries map to its
  platform section. IMPLEMENTATION defines acceptance for each of those areas.
  Deferred command examples are explicitly separated from release requirements.
- Review corrections: clarified bare-command precedence, unregistered inspection,
  identity changes, external concurrent personal edits, launch trust and partial
  failures. Trusted Unix launcher scripts need direct shebang execution; Windows
  profiles require native executables. No material product decision is pending
  for increment 1. Native OS behavior remains unverified.
- Coordination: no product clarification/proposal-approval questions; the pilot
  explicitly waived routine proposal review. Git branch creation required a tool
  permission escalation because `.git` is sandbox read-only; later Git permission
  outcomes are visible in the task transcript, separately from product decisions.
  No global configuration, policy revision or additional skill installation.
- Navigation walkthrough: README -> SPEC resource storage contract ->
  IMPLEMENTATION increment 3 acceptance; cross-feature entry continues through
  SPEC launch policy -> increments 2/4. This is a structural walkthrough only:
  implementation/tests do not exist. No duplicate command lists were added to
  AGENTS. Future implementation moves should update feature routes, not rewrite
  the root's internal file inventory. Later-task usability has not been measured.

### 2026-10-01: workspace metadata increment 1

- Task: continuation of the current Codex task, requested as "Continue with the
  plan." Policy remains v0.1; runtime model/settings and cost visibility are as
  recorded above. No model, global policy or skill changes.
- Start: `75c079dce321d6785d18c2adbed62f2d2e003170`, clean specification checkout.
  End: the implementation commit containing this entry on
  `feat/workspace-identity`; final task output records its hash and push result.
- Delivered: Go module and pinned dependencies; init/info/list/config commands;
  shared/device TOML validation; directory versus explicit workspace IDs; registry
  locking, backup and recoverable initialization; typed/JSON errors. Code and tests
  are routed through internal/README.md. Application launching, URL persistence
  and picker remain unimplemented, with explicit CLI errors.
- Technical refinement: init spans repository and external device filesystems.
  A small pending journal makes partial initialization recoverable without replacing
  a user-modified config. SPEC and README describe this and the hard-link filesystem
  requirement. The three storage scopes and source briefs are preserved.
- Actual local verification: the verification runner passed formatting, vet, tests
  and Windows/amd64 build using Go 1.27.0. Tests exercise fresh subprocesses,
  concurrent registrations, kernel lock release on process exit, interrupted file
  publication/init, preserved corruption, schema rejection, ambiguous names,
  symlink/case aliases, and checkout/storage separation. Both alias subtests ran
  successfully on this host. A built-binary smoke test used a disposable profile
  and checked init/info/list/config, persistence, repeat-init refusal and unsupported
  open. Linux/amd64 and macOS/arm64 cross-builds passed; these are not native tests.
- Verification limit: local race execution was attempted but unavailable because
  CGO is disabled and a C compiler is absent. CI config runs the same gate on
  Windows, macOS and Linux with an additional Linux race check; remote results are
  reported in the task output, not assumed here. No GUI behavior has been verified.
- Findings addressed: a malformed registry missing schema_version initially
  inherited a default; a regression test caught it and decoding now rejects it.
  Review also tightened storage overlap in both directions and clarified current
  versus planned commands. The first CI run passed native Linux/macOS gates and
  the Linux race check; Windows exposed checkout CRLF conversion before tests.
  A repository Go-file LF attribute corrects this without weakening gofmt checks.
  No product clarification was needed. Tool permission
  escalations for Git/cache/network access are separate from product approval;
  their actual outcomes are in the task transcript.
- Navigation: README -> internal/README -> workspace service/config/registry and
  their tests; CLI integration tests exercise the same service across processes.
  One verification runner serves local use and CI. No copied command inventory
  was added to AGENTS. Human review/rework effort, runtime cost, and usability for
  a fresh agent are not yet independently measured.

### 2026-10-02: Windows launcher increment 2

- Task: continuation toward the first usable Windows launcher. Policy v0.1 is
  unchanged; no global settings, dependencies or CI configuration were changed.
  Start: `83abb3354d236305411c1ca4bb1b025d3b8ed7ed`. End: the implementation
  commit containing this entry on `feat/windows-launcher`; final task output
  records the commit, push and remote check results.
- Delivered: read-only plans/digests, checkout-local approval, console consent,
  exact/script selectors, identity-change acceptance, Windows native editor,
  terminal, app and URL dispatch, per-resource results and MRU updates. README
  routes to ownership/tests through internal/README. Saved URLs and picker remain
  increment 3; other native adapters remain increment 4.
- Actual local gate: Go 1.27.0, Windows/amd64, `go run ./tools/verify` passed
  formatting, vet, tests and build. Tests cover unapproved/cancelled/read-only
  flows, target/profile changes, another worktree, re-preflight during approval,
  identity rebinding, corrupt trust, independent dispatch after failure, all-failed
  nonregistration, failed recency writes, concurrent entries and MRU ordering.
  CLI subprocess tests verify persisted consent across process restarts.
- Native helper tests: a detached delayed executable survived launcher exit and
  received exact arguments/CWD containing spaces, Unicode, leading punctuation,
  ampersands, semicolons, percent/dollar expressions, quotes and trailing slashes.
  Empty arguments survive; injection markers are absent. Default lookup rejects
  checkout executables and relative PATH entries. These are process tests, not
  evidence of GUI readiness.
- Desktop smoke: Windows 11 build 26200, VS Code 1.139.1, Windows Terminal package
  1.24.11911.0 (product 1.24.260710001), Firefox 157.0. A disposable workspace
  named `-Desky 雪 & sample; punctuation` used isolated device state/config,
  configured native Code with `--new-window`, default Terminal `new-tab -d .`,
  and `https://example.com/#desky-launcher-smoke`. All dispatches returned success;
  window enumeration showed the exact editor folder and Example Domain. Detailed
  computer-use inspection timed out awaiting app access. The user then explicitly
  confirmed all three opened correctly, including Terminal in the selected folder.
  Repeat entry reused approval and dispatched all three; info reported approved
  trust and count 2. The invoking shell's CWD was unchanged. Device files contain
  registry/trust/backup/lock only, with no process ownership or teardown records.
  Existing desktop apps were not closed or supervised.
  A further isolated run used actual default lookup with no device profiles:
  native Code `--reuse-window`, Terminal and URL all dispatched successfully;
  the editor folder and Example Domain window titles remained observable.
- Refinements: Terminal's own semicolon parser requires passing `.` through its
  argv and setting the child CWD natively. Re-preflight under the device lock
  detects recipe/profile changes during consent. Approval precedes dispatch;
  registration/recency follow at least one success. State failures retain per-item
  outcomes and warn against blind retry. Info remains usable with a trust-preflight
  diagnostic when a launcher or personal directory is unavailable.
- Limits: first-launch visible behavior was user-confirmed; repeat/default runs
  have dispatch and window-title evidence, not automated GUI focus assertions. No
  macOS/Linux desktop behavior is claimed. Local race execution still lacks a C
  toolchain; CI's Linux race job is separate evidence, reported after it runs.
- Coordination: one user observation request closed the native desktop gate after
  tool app-access timeout. No product/architecture decision or routine proposal
  approval was needed. Tool permission prompts are separate from product approval.
  Computer-use skill was used for window discovery. Human effort/cost beyond that
  observation was not independently measured. No generic pilot policy revisions.
- Remote verification found an additional Windows path-alias lookup case: a PATH
  entry was canonicalized but the supplied checkout root was not. The adapter now
  canonicalizes both before excluding checkout executables, with a symlink-alias
  regression test. Linux/macOS gates and Linux race passed on the first run;
  the corrected Windows result is reported in the task output.

### 2026-10-02: personal URLs and MRU selection increment 3

- Task: complete increment 3 while keeping unaffected resources opening during
  saved-URL conflicts and preserving conflicting data. Start:
  `acbcdeb8a3ec1fb3f2cae42c5d7e14531c647df0`. End: the implementation commit
  containing this entry on `feat/personal-urls`; final output records commit/push
  and remote verification. Policy v0.1 remains unchanged; no global settings,
  dependencies or CI changes. No agent delegation or product questions.
- Delivered: numbered MRU picker with query filtering and terminal-only prompts;
  personal URL add/list/pin/unpin/archive/restore; immutable revision history;
  exact-ID resource seam; explicit multi-head resolution; shared/personal pin
  deduplication and per-resource skipped results. SPEC explicitly replaces its
  original whole-entry conflict barrier as requested. Unknown/incomplete/corrupt
  per-resource data remains intact and visible; global enumeration failures still
  fail explicitly. Resource ownership and verification are routed in internal/README.
- Actual local checks: Go 1.27.0 on Windows/amd64; verification runner passed
  formatting, vet, all tests and executable build. Resource tests cover concurrent
  saves, status history, independent replica edits and union in different arrival
  orders, multi-head resolution, differing same-ID copies, provider-renamed files,
  missing parents, cycles, unknown schemas, malformed/duplicate JSON fields,
  unavailable storage and injected write failures. CLI subprocess tests verify
  fresh-process persistence, concurrent deduplication and safe output. Launch
  tests verify healthy dispatch during conflicts, preservation of every personal
  file, recency, pin deduplication and stable shared-recipe approval.
- Interactive smoke: Windows terminal/ConPTY via tool PTY, isolated profile and
  two disposable checkouts both named `Picker Demo`. Bare `desk --dry-run` showed
  both numbered paths; typing `beta` filtered to one row and `1` selected exactly
  that checkout, produced its plan/digest and exited. No application was dispatched.
  Automated tests additionally exercise Enter for top item, invalid numbers,
  filter clearing, cancellation, ambiguous names and JSON/redirected-input guards.
  This is observed terminal interaction, not a new GUI launch qualification.
- Performance: host `R-W11-MAIN`, Windows/amd64, AMD Ryzen 9 5900XT 16-Core,
  Go 1.27.0. `go test ./internal/launch -run '^$' -bench BenchmarkLargeWorkspace
  -benchtime=3x -count=1` measures local computation/filesystem access with 1,000
  real checkout directories and 1,000 current pinned URL resources; process
  startup, fixture creation and OS dispatch are excluded. Initial single-iteration
  list/dry-run were 78.6/89.9 seconds. Reusing directory FileInfo in registry
  validation reduced list to 142 ms; bounded parallel resource reads reduced dry
  run from 2.18 seconds to 320 ms. Final three-iteration means: list 135 ms,
  dry-run 320 ms. Both exceed the 100 ms aim; this is measured evidence, not a
  guarantee. No daemon or persistent cache was added, and alias/conflict checks
  remain enforced. Large externally synchronized histories may cost more.
- Limits/coordination: local race execution was attempted and Go rejected it
  because CGO is disabled; remote Linux race results are separate evidence.
  The shared Go build-cache permission error was avoided with a repository-local
  cache; build emitted a nonfatal module metadata cache-write warning. Git access
  uses tool permission escalation under standing task-branch authorization.
  No user-observation request was needed. Human review effort and runtime cost
  were not independently measured. Native macOS/Linux desktop qualification stays
  in increment 4.
- Initial remote CI passed Linux verification and the race check. Windows/macOS
  exposed a picker test comparing canonical selected paths with temporary-directory
  aliases. The fixture now uses the registered canonical paths, preserving the
  exact selection assertion. A further macOS run exposed a short filter query
  matching its random temporary ancestors under subsequence search; the filter
  assertion now uses the full target path and a distinct no-match query. Final
  remote results are reported in task output.

### 2026-10-02: adopt policy v0.2

- Adopted the generic v0.2 policy from autonomy-kit commit dbf831a, preserving
  Desky's project-specific reading and maintenance guidance. Start: 1745751.
  This entry's commit records adoption; prior evidence remains attributed to v0.1.
- Implementation now includes focused completion review, ownership of in-scope
  repairs, and verification of fixes. Review depth follows the change's risk;
  standalone review requests remain read-only.
- Scope is policy adoption only. The known first-URL-save recovery defect remains
  unfixed; no application code, product contracts, or global configuration changed.
  Document checks and policy comparison validate this adoption. No application
  tests or implementation trial of v0.2 are claimed.

### 2026-10-02: first-URL-save recovery repair

- Task: repair failed first-save/retry behavior and preserve existing/external
  personal data; stop after this repair. Policy v0.2 unchanged. Start:
  `f25201ff294fdc42e57b2e116644769313f2bab1`, clean working tree. End: the repair
  commit containing this entry on `fix/first-url-save-recovery`; final task output
  records its hash, push outcome and available remote checks. Exact model settings,
  runtime cost and measured human effort are unavailable; no global settings,
  dependencies, CI or pilot policy changes.
- Reproduced: the publisher created a resource directory before writing its first
  revision. An injected write failure left it empty; a successful retry created a
  second directory, while the first remained a permanent incomplete-data conflict.
  The regression failed in both a new directory workspace and one with saved data.
- Repair: track whether this publication created the resource directory and, on
  write failure, attempt only an atomic empty-directory removal. Never unlink a
  file or recursively delete contents. Preexisting directories remain untouched;
  any delivered file prevents removal. Late publication errors retain revisions,
  and retry uses existing URL deduplication. Registration remains stable without
  entry counts or trust. README/SPEC describe the recovery boundary and internal
  navigation points to the regression tests.
- Actual local verification: targeted resources/fileio tests and the full
  `go run ./tools/verify` gate passed on Go 1.27.0, Windows/amd64, including formatting,
  vet, all tests and executable build. The new tests recreate services before
  retry, verify usable status mutation, preserve earlier history and preexisting
  empty directories, and simulate provider copies, temporary delivery, a replaced
  directory and a published revision followed by an error. The build reported a
  nonfatal module metadata cache-write warning. Local race execution was attempted
  but Go rejected it because CGO is disabled; remote race evidence is separate.
- Limits: abrupt process termination before cleanup or a cleanup failure can
  still leave an incomplete directory. Such data remains visible for inspection;
  there is no sweeping repair of old or externally synchronized empty directories.
  This task does not implement increment 4 or qualify new desktop platforms.
- Completion review: a fresh reviewer inspected the full task diff, retry and
  deduplication paths, cleanup safety, external delivery and recovery docs; no
  actionable defects were found. The implementing agent also inspected the diff
  and integration paths. No product clarification or routine approval questions;
  Git/network sandbox escalations are separate tool permissions. Review used the
  personal commit-review-loop skill and one reviewer; no skill files were changed.

### 2026-10-03: increment 4 platform implementation and qualification gates

- Task: continue the next unfinished increment, implementing remaining native
  adapters and qualification checks. Policy v0.2 unchanged. Start: `4e6bd8f`,
  clean working tree. Task branch: `feat/native-platform-boundary`; the commit
  containing this entry records implementation. No installer, deployment,
  publication, merge, dependency or global configuration change.
- Implemented macOS/Linux executable lookup, explicit profiles and URL helpers;
  macOS Terminal default; required Linux terminal profile; new Unix process
  sessions and disconnected handles. Default Unix Code, open and xdg-open helpers
  have bounded dispatch waits. A timeout reports unknown outcome without killing
  children; completed children are reaped while the CLI remains alive.
- Native preflight rejects remote-session environment markers and detected
  headless sessions. Windows queries visible window-station flags, macOS checks
  local console ownership, Linux checks display environment. These detect session
  prerequisites rather than application health. Inspection and personal URL
  saving remain available in rejected launch sessions.
- The existing Unix flock/rename/fsync/hard-link persistence boundary needed no
  format or algorithm change. Existing concurrency, restart, conflict-preservation,
  recovery and storage-boundary tests remain native CI gates. Added Unix private
  directory/file/lock permission checks. Shared process tests now run on each OS,
  proving exact argv/CWD, no shell fallback, child survival and detached output
  handles. Fixture URL helpers exercise nonzero status, single URL argument and
  neutral home CWD; deadline tests prove a late helper can still finish.
- Local verification: Go 1.27.0, Windows 11 build 26200, amd64. Focused platform,
  CLI, fileio and launch tests passed; full formatting/vet/test/build gate passed.
  Cross-builds succeeded for Windows/macOS/Linux on amd64 and arm64. Unix test
  binaries were also compiled locally; none were executed as native tests here.
  The runner now exposes `-cross`; CI runs native suites on all three OSes and
  combines Linux race/cross checks in the same verification entry point.
- Local limits: race execution is unavailable because CGO is disabled. Builds
  report a nonfatal module metadata cache-write permission warning. The separate
  noninteractive Windows window-station fixture requires host privileges that
  were unavailable even outside the sandbox; that test explicitly skips on access
  denied. Invalid-station, simulated headless and SSH rejection checks still run.
  CLI dispatch subprocess tests require a visible Windows station; direct native
  child-process contracts run independently of GUI session availability.
- Completion review: a fresh reviewer found checkout-owned PATH directories
  linking to external launchers were accepted, default Unix Code helper failures
  were not awaited, and Windows lacked session preflight. All were repaired with
  regression coverage. The implementing agent owns validation and final inspection;
  the commit-review-loop skill was used without changing personal skills.
- Remote native CI: [run 37115845024](https://github.com/hollowhemlock/desky/actions/runs/37115845024)
  passed for implementation commit `4c0acbc`, using Go 1.27.0 on Ubuntu 24.04.5 LTS
  / amd64, macOS 26.6.2 (25G83) / arm64, and Windows Server 2025 (10.0.26100)
  Datacenter / amd64. All three native verification suites passed; Linux also
  passed the race detector and all six cross-builds. These are native helper,
  filesystem and domain test results. Nonverbose suite output does not establish
  whether optional desktop fixtures were skipped. The documentation follow-up
  `c3545c2` also passed all three jobs in
  [run 37116037278](https://github.com/hollowhemlock/desky/actions/runs/37116037278).
  The remaining release gates below keep increment 4 visibly open.

#### Windows desktop follow-up, 2026-10-03

- Host: Windows 11 build 26200 / amd64; VS Code 1.140.0, Windows Terminal package
  1.24.11911.0, Firefox 157.0, Notepad package 11.2607.14.0. The native Notepad
  launcher reports file version 10.0.26100.8457. The existing Windows Computer Use
  skill supplied application/window discovery and accessibility inspection.
- Fixture: `.cache/platform-smoke-20261003`, with a separate local device/personal
  directory and checkout named `-Desky increment 4 雪 & spaces; punctuation`.
  Recipe: native Code with `--new-window`, default Terminal, an intentional
  `https://example.com/#desky-increment-4-smoke` URL, and named app profile
  `smoke-notepad` opening the disposable `Desky increment 4.txt` document.
  Dry run displayed all four actions and did not dispatch. The actual console
  consent prompt accepted `y` and completed without hanging.
- Restricted-context entry produced real native failures: Terminal's WindowsApps
  alias returned "The file cannot be accessed by the system"; URL dispatch returned
  "No application is associated with the specified file for this operation".
  Desky reported `launch_failure`, attempted the independent Notepad action after
  the URL failure, and retained every result. Editor/Notepad process creation was
  acknowledged, but no fixture windows appeared in that context. This demonstrates
  why dispatched does not promise GUI readiness; no successful GUI behavior is
  inferred from those two process-creation acknowledgements.
- After inspecting those results and the desktop, an authorized run outside the
  restricted context reused the same approval. All four actions dispatched and
  returned control in approximately two seconds. Parent CWD was unchanged.
  Readback shows the same checkout/workspace IDs, approved trust and entry count 2
  (one partial entry and one fully dispatched entry). Device state contains only
  registry, backup, lock and trust files; no process ownership/teardown state.
- Subsequent desktop observation, after the CLI had exited, found the selected
  checkout in the Code window title, `Example Domain` in Firefox's window title,
  and the fixture document in Notepad. Notepad accessibility inspection verified
  the exact disposable document contents. Code accessibility inspection confirmed
  the checkout window; an existing extension reported a missing-ripgrep warning,
  which was left unchanged. These observations establish editor/named-app window
  survival after CLI exit, with browser evidence limited to its window title.
- Computer Use then explicitly stopped because URL-policy enforcement was not
  supported for this Windows browser. No browser-policy workaround was attempted.
  The skill also excludes terminal-app automation. The new terminal's visible CWD,
  browser page contents, and fully observed repeat-open behavior therefore still
  need human observation or an available supported inspection path. The earlier
  increment 2 user-confirmed smoke remains separate evidence, not a claim that
  these post-change checks were completed.
- The user confirmed no native macOS/Linux desktop hosts are currently available.
  Setup guidance was provided for an Ubuntu Desktop VM and a physical Mac with a
  logged-in desktop. No VM, host, software installation or remote access was
  provisioned. Native CI is complete; those desktop release gates remain open.

Actual desktop matrix and outstanding qualification:

| Host | Desktop evidence | Remaining gate |
|---|---|---|
| Windows 11 build 26200 / amd64 | Increment 2 user-confirmed smoke; increment 4 follow-up above adds actual consent, four-resource dispatch, native failure reporting, and editor/Notepad window survival and document inspection | Post-change visible terminal CWD, browser contents and fully observed repeat-open; real noninteractive-station fixture was denied on this host |
| macOS / architecture not yet exercised on a desktop | No desktop host available in this task | Exact OS/architecture/app versions; editor, Terminal directory with spaces, URL and named app; consent completion, repeat-open, return to CLI, failure cases and app survival |
| Linux / architecture not yet exercised on a desktop | No desktop host available in this task | Exact distro/session/architecture/app versions; editor, configured terminal, URL and named app; consent completion, repeat-open, return to CLI, failure cases and app survival |

On each newly available desktop, use isolated device/personal storage and a
disposable checkout with spaces and shell metacharacters. Record actual visible
directory/application behavior, prompt completion and repeat entry, parent CWD,
control returning promptly and survival after CLI exit. Run the native verification
runner, including recovery/concurrency/privacy tests, and record native launcher,
URL failure and absent-session outcomes. Do not substitute helper receipts or a
successful build for GUI observation. Public support/distribution remains a
separate decision and authorization.

To stop the local trial, remove only the generic pilot policy from AGENTS.md and
retain the project's reading and maintenance guidance. Start a fresh task. No
global Codex configuration needs to be rolled back.

### 2026-10-04: repeatable Ubuntu VM qualification tooling

- User-approved scope: Windows/PowerShell/VirtualBox tooling for a local Ubuntu
  24.04 Desktop amd64 ISO, reusable targets and committed-source transfer. Guest
  bootstrap runs once from the desktop with normal sudo prompts. No changes to
  Desky's application launch contracts, pilot policy or global configuration.
- Added `tools/vm` host lifecycle commands, private local overrides, bounded
  process invocation and credential-file cleanup, guarded vendor finalization,
  atomic source export/publication and exact source integrity checks. Existing
  VM accounts/hardware and unowned disks are preserved. Added idempotent guest
  prerequisites and explicit, individually observed qualification reports.
- The root README routes to the workflow's authoritative instructions and
  acceptance mapping. CI adds fake VirtualBox host tests and Linux Python/shell
  checks; no actual VM installation is inferred from those tests.
- Local host suite passes on PowerShell 7.6.6 / Windows, using a compiled fake
  VirtualBox executable and synthetic credentials. It exercises public invocation,
  settings, read-only status, reuse, interrupted configuration, ambiguous install
  failure, disk identity/collision refusal, guest argument preservation and
  committed-source exclusion of dirty/private/untracked files.
- Initial CI caught Git applying Windows line-ending preferences during archive
  export. A local regression reproduced it; export now disables those preferences
  for its Git invocation while retaining exact blob checks and explicit attribute
  rejection. Host tests cover both CRLF preferences and transforming attributes.
- An actual 73-file committed export was published and exactly verified on the
  WSL filesystem, including repeat publication. This checks cross-host source
  encoding and manifest interoperability, not VirtualBox transport or a desktop.
- Ten Python tests pass in the existing WSL Ubuntu environment. They cover
  checksums, atomic publication, concurrency, modified/missing/extra source,
  symlink refusal, generated-output allowances, non-dpkg editor reuse,
  observation status and failed account finalization using fake account commands.
  WSL was used only for filesystem/script tests, not desktop qualification.
- Full `go run ./tools/verify -cross` passed on Go 1.27.0 / Windows amd64,
  including all six compile targets. The first attempt could not write the
  global build cache; using the repository's ignored cache resolved it. A
  nonfatal module-version metadata-cache permission warning remained during
  builds. Shell syntax checks and `git diff --check` pass.
- Fresh independent review found a duplicated Guest Control executable argument,
  unsafe adoption of an existing disk path, and a metadata failure for existing
  non-dpkg VS Code installations. Repairs add correct argument dispatch,
  recorded medium UUID checks and separate executable-version reporting, with
  focused regressions. Review does not establish real installation success.
- Actual VirtualBox 7.2.20 read-only status reports the default target absent.
  Only Ubuntu 26.04.1 Desktop was found locally; its detection command returned
  an error despite some image metadata. No supported 24.04 ISO or real guest
  credentials were supplied, and no VM was created or installed. Actual ISO
  installation, Guest Control transport, repeated package bootstrap and visible
  desktop checks remain **unverified**. Increment 4 remains open.

### 2026-10-05: actionable ISO detection failures

- Reproduced the user's Ubuntu 26.04.1 ISO failure with VirtualBox 7.2.20.
  Detection returns exit 1 and `E_NOTIMPL` while still printing release metadata
  and `IsInstallSupported="on"`. The target VM remains absent; no installation
  or credentials were needed for reproduction.
- The helper now reports the detection failure, sanitized detected release and
  required Ubuntu 24.04 Desktop amd64 boundary before VM mutation. Missing
  metadata produces an actionable error instead of a strict-mode property error.
  Raw diagnostics remain suppressed. Documentation links to the supported image.
- Host regressions cover failed detection with supported/unsupported/empty
  metadata, successful detection of an unsupported release, missing fields,
  suppression of diagnostic details and absence of VM/state mutation. Successful
  fixture detection covers a four-component point-release version. These checks
  do not establish real Ubuntu installation or desktop qualification.
- Host regressions and the full Windows verification runner pass. Focused
  self-review and whitespace checks found no further defects in this repair.

### 2026-10-05: Ubuntu 26.04.1 target and partial detection

- The user selected Ubuntu 26.04.1 LTS. Host creation and guest bootstrap now
  accept Ubuntu 26.04 while retaining Ubuntu 24.04 compatibility. Hardware,
  credential handling and finalization requirements remain unchanged.
- Further investigation corrects the preceding interpretation of `E_NOTIMPL`:
  [VirtualBox's detector](https://github.com/VirtualBox/virtualbox/blob/master/src/VBox/Main/src-server/UnattendedImpl.cpp)
  converts incomplete Linux detection to that code, and its installer `prepare()`
  explicitly permits it. The prior exit-code-only guard was too strict. The helper
  now permits only the exact bare partial-detection diagnostic together with
  supported metadata and independent matching release/architecture/desktop boot
  information read from the ISO using Windows' built-in archive reader. Other
  errors carrying the same code are still rejected.
- Read-only validation of the local Ubuntu 26.04.1 amd64 ISO succeeds with
  VirtualBox 7.2.20. The vendor finalization template structure is also accepted.
  The detected `Ubuntu25_64` hardware profile is retained; the media's 26.04.1
  release is checked independently and recorded alongside the detection outcome.
- Host tests cover the real partial-result shape, complete detection, retained
  24.04 support, unsupported releases/architectures, added error details, missing
  metadata, inconsistent ISO contents and server boot entries. Eleven Linux
  guest tests pass, including normal-user/amd64 checks for both LTS releases.
  No actual installation, package bootstrap or desktop qualification is inferred
  from these checks. Installation still needs the user's local masked password
  prompt; no VM has been created by this task. Increment 4 remains open.
- Fresh review identified that older official Ubuntu ISO labels can have a
  numeric build-date respin suffix. The parser now accepts that convention,
  with a regression using the Ubuntu 24.04.1 label's `20240827.1` build date.
  The host suite and full Windows verification runner pass; the earlier nonfatal
  module metadata-cache permission warning remains unrelated to this change.

### 2026-10-05: use already-installed Ubuntu VMs

- The user reported that the installation remained at "begin loading essential
  drivers" and selected an existing-VM-only workflow. No cause of the boot stall
  was established. Automated OS installation is retired, superseding the previous
  installation plan and media-validation work; the stalled VM is preserved.
- Removed the create action, ISO/hardware creation settings, installer ownership
  state handling, vendor finalization/root-account changes and media cleanup.
  The helper retains status, harmless start, authenticated source provisioning and
  selected report collection. Ubuntu 26.04 and 24.04 Desktop amd64 remain accepted
  by guest bootstrap/qualification; detected live-installer boot modes are rejected.
- Existing VM names can include spaces and Unicode. Old creation settings produce
  a migration message. Private legacy records and installation files are neither
  read as prerequisites nor removed, including after a manual OS installation.
  Documentation explains selecting a target and retaining private installer files.
- Windows regressions exercise existing targets by name/UUID, missing and stopped
  targets, repeated start/provision, credential cleanup after authentication or
  transfer failure, Guest Additions failure, rejected create commands, and
  preservation of VM settings and old installer artifacts. The fake executable
  has no VM creation/configuration/installation commands. Linux checks retain
  exact source integrity and add rejection of live-installer sessions.
- These checks do not establish real Guest Control transfer, repeated package
  bootstrap or native desktop observations. Increment 4 remains open.
- Fresh review found that Ubuntu 26.04.1's normal live boot entry omits an explicit
  `boot=casper` option. Both host readiness and guest bootstrap now inspect the
  root filesystem as well, rejecting live/transient roots and missing root-mount
  evidence. Installed overlay-root systems are explicitly outside this workflow.
  The repair includes a regression using the actual ISO boot command and an
  overlay root; focused follow-up review found no remaining defects.
- The Windows host suite, twelve Linux guest tests, shell syntax check and full
  native verification runner pass. Only read-only status was exercised against
  real VirtualBox; no VM settings, disks, media or old installer files were changed.

### 2026-10-05: repair Guest Control copy destinations

- The user reported a generic `guestcontrol` exit 1 during provisioning. The
  local committed export and transport script were present; read-only host checks
  found `ubuntu-dev` running with Guest Additions 7.2.20 matching the host.
  Authenticated transfer could not be repeated without the user's local password
  prompt, so its original failing stage was not directly observed.
- Inspection found a concrete copy-target defect: VirtualBox's
  [GuestPath::BuildDestinationPath](https://github.com/VirtualBox/virtualbox/blob/main/src/VBox/Main/src-client/GuestCtrlPrivate.cpp)
  appends the source basename only when the destination ends in a separator.
  The CLI forwards `--target-directory` without adding one. The helper now
  normalizes the guest directory to end in `/`, retaining `--no-replace`.
- Provisioning now identifies staging, individual file copies and publication
  in progress/error messages without revealing raw guest diagnostics. Regression
  coverage exercises the real host transfer functions against the fake, rather
  than relying only on the lifecycle delivery stub. It covers all three copies,
  paths with spaces/Unicode/metacharacters, refusing overwrite, failures at each
  stage and preventing publication after a partial copy.
- Real transfer/bootstrap and observed desktop qualification remain open until
  the user retries provisioning and completes the guest-side workflow.
- The expanded Windows host suite and full native verification runner pass.
  The existing nonfatal Go module metadata-cache permission warning remains.

### 2026-10-05: diagnose guest publication replies

- The user's next run copied all three files and reached publication, which
  returned success but failed the host's exact revision-directory reply check.
  This confirms progress past the earlier copy failure; it does not establish
  verified publication or completed bootstrap.
- The exact exported archive, manifest and transport script published successfully
  through a Python subprocess on Linux-native temporary storage, returning the
  expected revision path with no stderr. An initial check on the Windows-mounted
  filesystem failed executable-mode integrity, so it was repeated on Linux-native
  storage without weakening mode checks. No guest credentials were accessed.
- Mismatch diagnostics now expose only output lengths and whether the expected
  complete path line is present. Raw guest output stays private, and unexpected
  or empty replies still fail. Host regressions cover those cases; a new Linux
  subprocess test covers the real CLI's exact reply, repeat publication and
  checksum failure, including spaces, Unicode and metacharacters in its path.
- The host suite, thirteen Linux tests and full native verification runner pass.
  The existing nonfatal Go module metadata-cache permission warning remains.
  The VM-specific cause remains unresolved pending the user's local diagnostic
  retry; no successful real publication or desktop qualification is claimed.

### 2026-10-05: confirm publication independently of process output

- The user's diagnostic retry returned exit zero with zero stdout and stderr
  characters. That establishes a missing reply, not successful source publication.
  VirtualBox's [guest process loop](https://github.com/VirtualBox/virtualbox/blob/main/src/VBox/Frontends/VBoxManage/VBoxManageGuestCtrl.cpp)
  completes on termination without a final output drain; lost final output is
  consistent with the observation, but has not been proven on this VM.
- Publication now optionally writes an exclusive `publication.json` receipt only
  after archive/tree verification and successful publication or verified reuse.
  The host retrieves that workflow file from the unique staging directory into
  the unique local export directory, then checks its schema, revision, destination,
  archive checksum and staging identity. Guest exit zero alone never confirms it.
  Existing receipts and symlink targets are preserved; a failed attempt may be
  retried with new staging, revalidating any already-published revision.
- Host regressions cover lost/noisy output with valid receipts and missing,
  malformed, mismatched or unavailable receipts. Linux subprocess tests discard
  stdout, verify the receipt, reject existing/linked receipts and prove checksum
  or unexpected-source failures produce no receipt. The Windows host suite,
  fifteen Linux tests, guest shell syntax check and full native verification pass
  (the existing nonfatal Go module cache permission warning remains). Independent
  review found no actionable defects. A real Guest Control retry, bootstrap and
  desktop qualification remain open; no VM credentials were retained or accessed.
