# Desky autonomy pilot

Status: the first architecture/specification task is documented in
[SPEC.md](SPEC.md) and [IMPLEMENTATION.md](IMPLEMENTATION.md). Workspace metadata
and Windows entry, personal URL persistence and selection (increments 1-3) are
implemented. Browser integration remains planned. Evidence appears below.

Baseline: `97f7edb443fde605932fc08378e6021b403d50cd` contains the original README,
product brief, and terminal namespace design. The policy is the v0.1 snapshot in
[AGENTS.md](AGENTS.md). This trial uses the existing Codex tools and permissions;
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
  exact selection assertion. Final remote results are reported in task output.

To stop the local trial, remove only the generic pilot policy from AGENTS.md and
retain the project's reading and maintenance guidance. Start a fresh task. No
global Codex configuration needs to be rolled back.
