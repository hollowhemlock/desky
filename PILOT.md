# Desky autonomy pilot

Status: local guidance prepared. No application task, behavioral evaluation, or
cross-platform execution has been completed by this setup.

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

To stop the local trial, remove only the generic pilot policy from AGENTS.md and
retain the project's reading and maintenance guidance. Start a fresh task. No
global Codex configuration needs to be rolled back.
