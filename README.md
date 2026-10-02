# Desky

A cross-platform project-context launcher for fast re-entry into a working
directory and its editor, terminal, applications, and saved web resources.

Current state: architecture and first-release specification completed. The
selected stack is a Go CLI with TOML configuration and local file persistence.
Application code, toolchain manifests, and executable verification have not been
added. All commands described in the specification are planned, not available.

## Start here

| Document | Role |
|---|---|
| [AGENTS.md](AGENTS.md) | Required reading order, maintenance rules, and the project-local autonomy pilot policy |
| [plan.md](plan.md) | Product intent, boundaries, identity, storage scopes, privacy, and requested next deliverable |
| [namespace.md](namespace.md) | Proposed terminal interface, command semantics, and first-version versus future scope |
| [SPEC.md](SPEC.md) | Selected release boundary, architecture, identity, schemas, CLI, trust, platform launch and browser seam |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | Incremental acceptance, planned verification, Git integration context and readiness review |
| [PILOT.md](PILOT.md) | Ready-to-use first-task prompt and evaluation sequence |

The source documents mix firm constraints with examples and possible future
features. Preserve their distinctions when deriving the specification. Update
this entry point as concrete specifications and implementation become available;
point to their authoritative sources instead of maintaining parallel inventories.

For implementation, start with increment 1 in IMPLEMENTATION.md and read the
affected contracts in SPEC.md. No application installation or test command works
yet. As each increment lands, add routes here to its real implementation and
verification entry point; retain this distinction between planned and working.
