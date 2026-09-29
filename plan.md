We are designing a cross-platform developer workspace/context launcher.

## Problem

Virtual desktops create too much hidden state and make it easy to lose attention/context, especially browser tabs.

The primary goal is extremely fast re-entry into a project.

Think:

    cd ~/src/project
    work

or:

    work project-name

The tool should restore the user's *working context* without becoming a process manager, virtual desktop manager, environment manager, or cloud synchronization service.

A useful analogy is:

    direnv / mise:
        directory → computational environment

    this tool:
        directory → human working environment

## Core responsibility

`work` should handle things such as:

- opening/focusing the editor for a directory
- opening/focusing a terminal for the directory
- opening project URLs
- opening desktop applications
- providing a recent/frecency-based project picker
- remembering intentionally saved web resources
- optionally exposing explicit launch actions

It should NOT own:

- environment variables
- runtime/tool versions
- PATH
- package environments
- Docker/devcontainer lifecycle
- long-running processes by default
- process teardown
- application shutdown
- virtual desktops
- user authentication
- hosted synchronization

Existing tools such as mise, direnv, Nix, Dev Containers, Docker, etc. should continue owning execution/environment concerns.

## Lifecycle

Use an `open` model, not a managed `start/stop` session model.

Conceptually:

    workspace.open()

Avoid:

    workspace.start()
    workspace.stop()
    workspace.isOpen

There should be no requirement to reliably tear down what was opened.

Repeated `open()` should be safe/idempotent-ish where practical:

- reuse an editor/project if the application supports it
- activate existing applications when appropriate
- otherwise rely on normal OS/application launch semantics

Do not build complicated process/window ownership tracking solely to guarantee idempotence.

## Workspace discovery / entry

The directory is the primary entry point.

Examples:

    cd repo
    work

    work ~/code/foo

    work foo

The no-argument/global interface should provide a very fast MRU/frecency picker.

Something like:

    > payments-api      ~/src/payments-api      4 min ago
      website           ~/src/website           yesterday
      infra             ~/src/infra             3 days ago

Recency should probably be weighted strongly.

Tools such as zoxide and `pj` are useful precedents for frecency/project discovery.

## Workspace identity vs checkout identity

A directory is the entry point, but a Git project can have multiple checkouts/worktrees.

Preferred model:

    project/workspace = durable logical identity
    checkout directory = workspace instance

For example:

                payments-api
                workspace ID
                    |
             +------+------+
             |             |
      ~/src/payments   ~/src/payments-fix
       main checkout     worktree

Durable resources such as saved documentation should belong to the logical workspace.

Instance-specific things such as the current path belong to the checkout.

Do NOT infer identity solely from Git remote URLs.

Prefer an explicit stable ID in committed configuration:

    [workspace]
    id = "..."
    name = "payments-api"

Without an explicit workspace identity, directory identity should win rather than using clever Git heuristics.

## Repository configuration

Keep the committed repository footprint small.

Example:

    repo/
      workspace.toml

`workspace.toml` is tracked and can contain:

- workspace ID
- workspace name
- shared URLs
- shared entry points
- shared launch definitions

Example:

    [workspace]
    id = "..."
    name = "payments-api"

    [[resource]]
    type = "url"
    name = "API docs"
    url = "https://..."

    [[resource]]
    type = "editor"
    path = "."

Potentially support an optional local override:

    workspace.local.toml

This would be globally gitignored and intended for machine/check-out-specific configuration such as preferred editor or machine-specific executable paths.

Do not put the main personal/session database in a gitignored folder inside each repo.

## Storage scopes

There should be three clearly distinct storage scopes.

### 1. Repository/shared

Committed through Git.

Contains:

- workspace ID
- shared workspace definition
- shared project URLs
- standard launch recipe

### 2. Personal persistent

Outside the repository.

Contains intentional personal state such as:

- saved web pages
- personal pins
- aliases
- perhaps personal notes/preferences

The application itself does NOT sync this.

The user can configure the data directory to live inside:

- Dropbox
- Google Drive
- OneDrive
- Syncthing
- another filesystem synchronization system

For example:

    personal_data_dir = "~/Dropbox/Workspaces"

The program simply reads/writes files.

No accounts.
No OAuth.
No credentials.
No hosted backend.
No authentication service.

Intentional persistent data is expected to follow the user across machines when they place this directory in an externally synchronized folder.

Do not call this feature "our sync"; it is just a configurable personal-data directory.

### 3. Device-local/session

Stored in the OS-specific application data/state location.

Contains observational/ephemeral state such as:

- last browser tabs
- window IDs
- current browser-session state
- machine-specific MRU details if appropriate
- checkout paths
- other transient data

This should not intentionally synchronize.

Principle:

    deliberate state → persistent / can follow user
    observed state   → device-local

## Browser integration

Browser context is important.

Fixed URLs are easy, but users should also be able to save the page they are currently viewing without manually editing workspace.toml.

A Firefox extension is likely appropriate.

Potential interaction:

    Save to payments-api
    Save + pin

Keyboard shortcut would also be useful.

The extension should talk to the local workspace tool/service; the browser extension should not become the canonical workspace database.

Future Chrome/etc. extensions could use the same local interface.

### Page states

Each saved page remembers its status.

Suggested model:

- Pinned
  - explicitly saved
  - durable
  - normally opened/surfaced on workspace entry

- Saved
  - explicitly saved
  - durable
  - associated with workspace
  - not necessarily opened automatically

- Archived
  - explicitly retained but removed from normal view

- Last tabs/session
  - optional observational browser state
  - device-local
  - separate from saved resources

Nothing intentionally saved should automatically expire.

## Browser privacy

Automatic browser capture must be OPT IN.

Default behavior is manual save only.

Important principles:

- never automatically capture private/incognito windows
- automatic session capture off by default
- show clearly when capture is active
- allow easy pause
- potentially allow excluded domains
- automatically observed tabs must NOT silently become permanent saved resources

A possible automatic mode is "Workspace Window":

- user explicitly designates a browser window as belonging to a workspace
- tabs in that window can be remembered as session state
- session data remains separate from explicitly saved pages

This avoids accidentally turning the product into a browsing-history recorder.

If browser extensions synchronize any browser-specific state, they should use the browser vendor's sync mechanism rather than an authentication/sync service provided by this project.

## CLI ideas

Possible commands:

    work
        show/select recent workspace

    work .
        open workspace for current directory

    work foo
        fuzzy-resolve workspace/project

    work -
        possibly reopen most recent workspace

    work add-url <url>
        save URL to current workspace

    work add-url --clipboard
        save current clipboard URL

Potential browser-extension equivalents should exist for URL saving.

Long-running actions/tasks should remain optional.

For example, do NOT implicitly run `npm dev` unless explicitly configured/requested.

Environment/task systems such as mise may already own that responsibility, so avoid conflicts.

## Design principles

1. Optimize for entry speed.
2. Directory is the primary entry point.
3. Project is durable identity; checkout can be an instance.
4. Restore attention/context, not OS desktop state.
5. Launch rather than manage lifecycle.
6. No teardown requirement.
7. Don't duplicate direnv/mise.
8. No hosted service.
9. No application-managed user accounts.
10. No authentication/credentials.
11. Let Git synchronize shared project config.
12. Let filesystem-sync tools synchronize personal persistent data.
13. Keep session/observational data device-local.
14. Browser auto-capture is explicitly opt-in.
15. Keep the first implementation small.

## Existing precedents considered

- Bunch: context/workspace launching, but macOS and lifecycle-oriented.
- PowerToys Workspaces: Windows app/window restoration and reconciliation.
- `pj`: project index + frecency + editor launching.
- zoxide: excellent precedent for directory frecency.
- Dev WorkDir: recent directory/project launcher on macOS.
- Warp Launch Configurations: declarative startup, terminal-centric.
- Tmuxinator: repo-associated declarative terminal sessions.
- direnv / mise: useful conceptual precedent for directory-scoped behavior, but environment management is deliberately outside this project's responsibility.

## Product shorthand

A useful description is:

"zoxide for workspaces"

or more precisely:

"Select a recent project directory, execute its human-context launch recipe, and get out of the way."

## What I want from Codex next

Start by turning this into a concrete architecture/spec rather than immediately overbuilding it.

Please:

1. Inspect the current repository if one exists.
2. Propose the smallest viable cross-platform architecture.
3. Define the configuration/state schemas and storage locations.
4. Define workspace vs checkout-instance identity.
5. Define the CLI surface.
6. Identify the minimal OS abstraction needed for macOS, Windows, and Linux app/URL/terminal launching.
7. Keep browser integration behind a narrow interface so a Firefox extension can be added separately.
8. Avoid building sync, authentication, teardown, environment management, or complicated window management.
9. Point out unresolved product decisions before implementing assumptions that will be expensive to reverse.
10. Then propose an incremental implementation plan.