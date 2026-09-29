# Terminal Namespace Design

## Purpose

This document defines the terminal-facing namespace and command structure for the project.

The project is a fast project-context launcher. Its CLI should optimize for:

- extremely fast entry into a project
- minimal typing for common operations
- predictable behavior
- strong separation between workspace selection and workspace configuration
- room for browser/editor integrations without polluting the primary namespace
- compatibility with shells, scripts, aliases, and fuzzy launchers
- avoiding ownership of unrelated concerns such as environment management, process supervision, or synchronization

The terminal interface should feel closer to `zoxide` or `git` than to a desktop workspace manager.

---

# 1. Naming Principles

The executable name should be:

- short
- easy to type
- easy to pronounce
- usable as both a noun and command where possible
- unlikely to collide with common Unix/Windows commands
- not imply process/session ownership
- not imply environment management
- not imply that everything being opened will later be closed by the tool

Potential names currently include:

- `desk`
- `focus`
- `resume`
- `ctx`
- `enter`
- `jump`
- `recall`

Examples in this document use `desk`.

The final executable name should be considered replaceable until implementation stabilizes.

---

# 2. Primary Interaction

The most important command is the bare command:

```sh
desk
```

It should open a fast recent/frecency-based workspace selector.

Example:

```text
> payments-api       ~/src/payments-api       8m
  website            ~/src/website            1d
  infrastructure     ~/src/infra              3d
```

The next most important form is:

```sh
desk .
```

This means:

> Open the workspace associated with the current directory.

And:

```sh
desk foo
```

means:

> Resolve `foo` against known workspaces and open the best match.

The root command therefore doubles as both:

1. workspace selection
2. workspace opening

This keeps the most frequent path extremely short.

---

# 3. Positional Namespace

The first positional argument should normally be interpreted as a workspace selector.

Examples:

```sh
desk payments
desk ~/src/payments
desk .
desk ..
```

Resolution can include:

- exact workspace name
- alias
- path
- current directory
- fuzzy workspace-name match
- fuzzy path match

The command should avoid requiring:

```sh
desk open payments
```

for normal use.

That form is unnecessarily verbose for the dominant interaction.

`open` may still exist as an explicit command for scripting or clarity.

---

# 4. Special Selectors

A small number of symbolic selectors are useful.

## Current directory

```sh
desk .
```

Open the current workspace.

## Previous workspace

```sh
desk -
```

Open the previously entered workspace.

This follows familiar shell behavior such as:

```sh
cd -
```

## Explicit path

```sh
desk ~/code/foo
```

Resolve the workspace from the supplied directory.

Paths should take precedence over fuzzy name matching when they resolve to an existing filesystem location.

---

# 5. Explicit `open` Command

An explicit version should exist:

```sh
desk open foo
```

It should behave approximately the same as:

```sh
desk foo
```

Why keep it?

- scripts are more readable
- third-party integrations can avoid ambiguous positional parsing
- future CLI extensions have a stable explicit API
- browser/editor integrations can call a semantically precise command

Human interactive use can remain:

```sh
desk foo
```

while machines can prefer:

```sh
desk open --workspace foo
```

---

# 6. Workspace Discovery Namespace

Commands relating to known workspaces should live under a single namespace.

Suggested:

```sh
desk list
desk recent
desk find <query>
```

## `list`

Lists known workspaces.

```sh
desk list
```

Possible output:

```text
payments-api    ~/code/payments
website         ~/code/website
infra           ~/code/infra
```

This command should favor scriptability.

For example:

```sh
desk list --json
```

## `recent`

Explicit MRU/frecency view:

```sh
desk recent
```

The bare:

```sh
desk
```

may simply be the interactive equivalent of `desk recent`.

## `find`

Resolve/search without opening:

```sh
desk find payments
```

Useful for shell integrations and debugging workspace resolution.

---

# 7. Initialization Namespace

Creating project configuration should be explicit.

```sh
desk init
```

Run in a project directory:

```sh
cd ~/code/foo
desk init
```

Possible result:

```text
Created workspace.toml
Workspace: foo
ID: 01K...
```

Useful options:

```sh
desk init --name foo
desk init --no-git
```

Initialization should not create large state directories inside the repository.

The primary repository artifact should remain something like:

```text
workspace.toml
```

---

# 8. Configuration Namespace

Use:

```sh
desk config
```

for global configuration.

Examples:

```sh
desk config get editor
desk config set editor code
desk config set personal-data-dir ~/Dropbox/Desk
desk config path
desk config edit
```

Project configuration should generally remain file-based:

```text
workspace.toml
```

There should be no need for a complex secondary configuration database.

A convenience command may exist:

```sh
desk config project
```

to show which configuration file applies to the current directory.

---

# 9. Resource Namespace

Resources are durable things associated with a workspace.

Examples:

- URLs
- editor targets
- documentation
- desktop applications
- project dashboards
- filesystem paths

Use a `resource` namespace:

```sh
desk resource list
desk resource add
desk resource remove
```

However, frequently used resource types should receive convenience aliases.

For example:

```sh
desk url add https://example.com
```

can internally be equivalent to:

```sh
desk resource add --type url https://example.com
```

This keeps the internal model generic without making everyday operations verbose.

---

# 10. URL Namespace

URL management is important enough to deserve a small first-class namespace.

Examples:

```sh
desk url add https://docs.example.com
desk url list
desk url remove <id>
```

When run from inside a workspace:

```sh
desk url add https://docs.example.com
```

the current workspace should be inferred.

Options:

```sh
desk url add --pin https://docs.example.com
desk url add --title "Payments API" https://docs.example.com
desk url add --clipboard
```

Possible status operations:

```sh
desk url pin <id>
desk url unpin <id>
desk url archive <id>
desk url restore <id>
```

A browser extension can map directly onto these concepts without invoking the CLI literally.

---

# 11. Save Namespace

A broader convenience action may be desirable:

```sh
desk save
```

The meaning should be:

> Persist something intentionally into the current workspace.

For example:

```sh
desk save url https://...
desk save path ./notes.md
```

However, avoid making `save` a vague catch-all if resource-specific commands remain clearer.

The browser extension's "Save to workspace" language does not require the terminal command itself to be named `save`.

---

# 12. Workspace Information

Use:

```sh
desk info
```

to inspect the workspace associated with the current directory.

Example:

```sh
desk info
```

Output:

```text
Workspace: payments-api
ID: 01K...
Path: /Users/me/code/payments-api
Config: /Users/me/code/payments-api/workspace.toml

Pinned:
  Stripe Docs
  localhost

Saved:
  Retry RFC
  PR #421
```

For another workspace:

```sh
desk info payments
```

This is useful both for humans and debugging.

---

# 13. Checkout / Instance Namespace

The distinction between logical workspace and checkout instance should generally remain invisible during normal use.

Users should not have to type:

```sh
desk instance open
```

for everyday work.

However, diagnostic commands may expose it.

For example:

```sh
desk checkout list
```

or:

```sh
desk instance list
```

Possible output:

```text
payments-api

* ~/code/payments
  branch: main

  ~/code/payments-hotfix
  branch: hotfix/refunds
```

`checkout` is probably more understandable than `instance` for Git-oriented projects.

Internally, "instance" may remain the implementation terminology.

---

# 14. Personal Data Namespace

The application should not expose synchronization as though it were a service.

Avoid commands such as:

```sh
desk sync
desk login
desk account
desk cloud
```

The project does not provide any of those services.

Instead:

```sh
desk config set personal-data-dir ~/Dropbox/Desk
```

can determine where persistent personal state is stored.

A diagnostic command could show this:

```sh
desk data
```

or:

```sh
desk data path
```

Example:

```text
Personal:
  /Users/me/Dropbox/Desk

Device state:
  /Users/me/Library/Application Support/Desk
```

Commands might include:

```sh
desk data path
desk data personal
desk data local
```

The word `sync` should be avoided unless referring explicitly to an external provider.

---

# 15. Session Namespace

Session state is secondary and should not dominate the product terminology.

Possible commands:

```sh
desk session show
desk session clear
```

This could cover:

- remembered browser tabs
- temporary browser-window associations
- ephemeral launch observations

The command:

```sh
desk session clear
```

should never delete deliberately saved or pinned resources.

That boundary must remain strong.

---

# 16. Browser Namespace

Browser integration may justify:

```sh
desk browser
```

but most interaction should happen through browser extensions.

Potential diagnostic commands:

```sh
desk browser status
desk browser sessions
desk browser clear-session
```

Avoid making browser-specific concerns leak into the root namespace unless they are frequently needed.

For example, prefer:

```sh
desk browser clear-session
```

over:

```sh
desk clear-tabs
```

---

# 17. Actions and Tasks

The project should distinguish opening context from running optional actions.

If workspace definitions expose optional commands, use:

```sh
desk run <action>
```

Examples:

```sh
desk run dev
desk run tests
desk run database
```

This should be explicitly invoked.

Running:

```sh
desk
```

should not automatically imply:

```sh
npm run dev
docker compose up
```

unless the project configuration very explicitly opts into such launch behavior.

Even then, the design should defer long-running development-process orchestration to existing tools where practical.

The namespace should avoid becoming another task runner.

---

# 18. Recommended Root Namespace

A minimal first version could expose only:

```text
desk
desk <workspace>
desk .
desk -

desk open
desk init
desk list
desk recent
desk info

desk url
desk config
desk run
```

Everything else can be added later.

That is intentionally small.

---

# 19. Proposed CLI Tree

A mature version might eventually look like:

```text
desk
├── <workspace>
├── .
├── -
│
├── open
├── init
├── list
├── recent
├── find
├── info
│
├── url
│   ├── add
│   ├── list
│   ├── pin
│   ├── unpin
│   ├── archive
│   └── remove
│
├── resource
│   ├── add
│   ├── list
│   └── remove
│
├── run
│
├── checkout
│   └── list
│
├── session
│   ├── show
│   └── clear
│
├── data
│   └── path
│
└── config
    ├── get
    ├── set
    ├── path
    └── edit
```

This is a possible eventual namespace, not a requirement for the first release.

---

# 20. First-Version Recommendation

The first release should probably implement only:

```sh
desk
desk .
desk <query>

desk init
desk list
desk info

desk url add
desk url list

desk config
```

Possibly:

```sh
desk -
```

if previous-workspace tracking is already available.

Everything else should wait until there is demonstrated need.

---

# 21. Shell Integration

Shell integration should remain optional.

Useful additions could include:

```sh
eval "$(desk shell-init zsh)"
```

or conventional shell completion installation.

Potential capabilities:

- tab completion for workspace names
- shell-aware directory switching
- fuzzy picker integration

Do not require a shell hook merely to use the basic application.

A user should always be able to run:

```sh
desk foo
```

without modifying shell startup files.

---

# 22. Shell Directory Changes

One architectural constraint deserves special attention:

a child process cannot normally change the working directory of its parent shell.

Therefore, if the desired behavior eventually includes:

```sh
desk foo
```

followed by the current shell itself landing in:

```text
~/code/foo
```

that requires one of:

- a shell function
- an emitted command evaluated by the shell
- shell-specific integration

For example:

```sh
desk path foo
```

could return:

```text
/Users/me/code/foo
```

and shell integration could implement:

```sh
d() {
    cd "$(desk path "$@")"
    desk open .
}
```

The core application should not pretend it can portably mutate its parent shell.

Launching applications for the target directory does not require this integration.

---

# 23. Machine-Friendly Interface

Anything likely to be consumed by extensions, scripts, or editor plugins should have a structured-output form.

Examples:

```sh
desk list --json
desk info --json
desk find foo --json
```

However, external integrations should preferably communicate through a narrow local API rather than continuously shelling out.

The CLI and local API should use the same underlying domain model.

---

# 24. Error Semantics

The CLI should distinguish clearly between:

- workspace not found
- ambiguous workspace
- invalid configuration
- launch failure
- resource failure
- unsupported platform action

For example:

```text
$ desk foo

Ambiguous workspace "foo":

  foo-api       ~/src/foo-api
  foo-web       ~/src/foo-web

Run `desk foo-api`, or select interactively.
```

Interactive terminals may prompt.

Noninteractive/scripted execution should fail rather than making a fuzzy guess when confidence is low.

---

# 25. Avoid Namespace Bloat

Avoid adding commands merely because the internal model contains an object.

The CLI does not need to expose concepts such as:

```text
workspace-manager
process-owner
window-reconciler
sync-provider
authentication
environment
runtime
daemon-manager
```

unless those become actual product responsibilities.

The root namespace should describe things users think about, not internal implementation details.

---

# 26. Recommended Semantic Model

The CLI should communicate this mental model:

```text
desk foo
    ↓
resolve project
    ↓
resolve checkout
    ↓
load shared + personal context
    ↓
open relevant resources
    ↓
return control to user
```

Not:

```text
start managed session
    ↓
own applications
    ↓
track entire desktop
    ↓
eventually tear everything down
```

The command namespace should reinforce that distinction.

---

# 27. Current Preferred Shape

The most promising interaction currently looks like:

```sh
desk
```

Pick something recent.

```sh
desk .
```

Open this project.

```sh
desk payments
```

Jump directly to a known project.

```sh
desk -
```

Return to the last project.

```sh
desk url add --clipboard
```

Remember something useful.

```sh
desk info
```

See what belongs to the current project.

Everything else should be considered secondary.