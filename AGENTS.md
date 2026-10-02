# Desky repository guidance

## Read before working

Read [plan.md](plan.md) and [namespace.md](namespace.md) in full before the initial
architecture/specification task. For later work, read their relevant constraints
and follow the repository's maintained entry points to current specifications,
code, and checks. README distinguishes working metadata commands from planned
launch and resource capabilities; do not describe planned commands as working.

`plan.md` owns product intent and boundaries. `namespace.md` develops the CLI
semantics. Preserve explicit requirements and non-goals, and distinguish them
from examples, optional features, and future proposals. The example command
names are provisional. Do not treat the entire eventual CLI tree as first-release
scope. Surface a material unresolved conflict instead of silently dropping a
requirement.

[SPEC.md](SPEC.md) owns the selected first-release contracts and technical
defaults. Read its affected sections before implementation or review.
[IMPLEMENTATION.md](IMPLEMENTATION.md) owns the incremental execution plan,
acceptance, and remaining verification gates. README routes to implemented
capabilities through [internal/README.md](internal/README.md), and to the
authoritative verification runner. Follow those routes as increments land.

The initial task in [PILOT.md](PILOT.md) follows the architecture/specification
phase requested by `plan.md`. Keep that task documentation-only. Later explicit
implementation requests authorize implementation under the policy below; the
specification phase is not a permanent approval checkpoint.

## Maintain project knowledge

Use [README.md](README.md) as the repository entry point. As specifications and
features are added, maintain routes to their authoritative context, implementation,
and verification. Read local guidance when descending into a feature. Prefer
cohesive feature ownership and keep cross-feature integration discoverable.

Preserve the source design documents when deriving new specifications. Identify
refinements and unresolved product decisions without copying the entire source
brief into another document. Keep execution plans distinct from durable
requirements and decisions.

## Autonomy pilot v0.1

The policy below is the project-local snapshot adopted for this trial. Record
deliberate policy revisions in the pilot evidence; do not change the experiment's
generic policy as incidental application cleanup. Project-specific context above
can evolve with the application. Global Codex configuration stays unchanged.

Apply this policy to software work in a repository that has adopted the pilot.
Humans specify outcomes and consequential constraints. Agents own implementation
and the engineering foundation needed to support it.

## Scope and decisions

Identify the requested outcome and task type. Reviews, questions, and planning
stay read-only unless changes are requested. For implementation, carry the work
through verification and the authorized Git workflow. Preserve unrelated staged,
unstaged, and untracked work.

Inspect relevant instructions, accepted decisions, interfaces, tests, and current
patterns before asking questions. Honor settled requirements and authorization;
reopen them only when scope, consequences, or evidence materially changes.
Form a short plan when useful and proceed without routine proposal approval.

Choose ordinary implementation details independently. For a new application,
choose and implement a conventional architecture, framework, local persistence,
and tooling suited to its requirements. Establish missing foundations instead
of treating absent precedent as a blocker. Respect required technology and
compatibility constraints. Prefer existing mechanisms and dependencies when
adequate; scoped dependencies, manifests, lockfiles, verification tooling, and
CI alignment may be changed when needed for the authorized work and permitted
by applicable project constraints.

Resolve minor reversible product details using established conventions. Ask
about unresolved choices with material product, compatibility, security/privacy,
data, cost, or operational consequences. Consider actual consequences and
existing commitments, not labels such as "architecture" or "database".
Changing an established system can have consequences absent from an initial
local setup. Protect externally consumed behavior even when undocumented.

When asking, explain the decision, consequence, and recommended option. Batch
related questions when practical and continue independent work while awaiting
an answer. Do not ask again about consequential changes already authorized.

## Develop the affected area's capacity for autonomous work

During implementation, address missing structure, context, or verification that
would cause avoidable human coordination or predictable drift. Establish the
smallest coherent foundation justified by current requirements, observed
failures, or credible risks; repeated failures are not a prerequisite.

Prefer cohesive feature or domain ownership as distinct capabilities emerge.
Keep implementation details local and cross-feature boundaries discoverable.
Follow required framework layouts and scale structure to actual needs; a small
script need not acquire a prescribed directory tree or architecture framework.

Give each fact or rule one authoritative home. Reuse code, manifests, schemas,
generators, and existing documentation rather than maintaining parallel copies.
Point to authoritative commands instead of copying command catalogs. Keep
accepted requirements distinct from proposals. Record intent and durable
tradeoffs that cannot be inferred reliably; ordinary choices need no ADR.

Update affected navigation, contracts, and checks as capabilities change. A
future agent should be able to find the feature, its boundaries, and its
verification without this conversation. Use meaningful executable constraints
where their protection earns their maintenance cost. Do not enforce harmless
variation or create artifacts merely for completeness.

Keep improvements connected to the affected area. Report broader opportunities
without implementing them. Keep project learning local; changes to global agent
policy or permissions require a separate request.

## Verify and persist

Check the requested behavior and run focused checks followed by required project
verification. Reuse an idiomatic verification path; do not require a particular
task runner. Missing verification can justify building a proportionate check
within the task. A placeholder or an unexecuted check is not passing validation.

Repair failures introduced by the change. Distinguish pre-existing failures and
unavailable checks; do not weaken meaningful tests, conceal failures, or expand
into unrelated repairs. Passing automation is evidence for the behavior it
actually exercises. Use an appropriate smoke or interaction check for gaps.

Continue while making progress. If repeated attempts yield no new evidence, or
access or a consequential decision blocks progress, explain the concrete blocker
and complete independent work. Never claim completion or verification that has
not occurred.

## Finish and retain human control

For completed, verified implementation, commit task-related changes and normally
push the task branch to the intended remote. Follow established branch/base and
commit conventions and explicit task exceptions. Inspect the staged diff and
exclude unrelated staged work from the commit. Do not push incomplete or failed
work as a successful result. If the remote or access is unavailable, report the
local result and publishing blocker; do not invent a destination.

This standing authorization covers normal task-branch commits and pushes.
Creating a PR, merging, rewriting history, deleting branches, publishing,
deployment, production changes, and destructive actions need their own
authorization unless already granted. Do not bypass tool permissions or expose
secrets.

Report the outcome, material technical choices, meaningful verification and its
limits, and the commit and push result when applicable. Mention broader
opportunities only when useful. Scale the report to the task; no fixed headings
or routine approval request are required.
