---
name: manage-local-secrets
description: Manage local development secrets through the VCSM protected broker without observing values. Use when an agent needs to create, rotate, revoke, list, audit, or use named secrets through pre-approved actions in Codex, Claude Code, Cursor, or another local coding workflow.
---

# Manage Local Secrets

Treat every secret value and the VCSM master password as non-observable. Operate only with project, environment, secret name, action name, version, state, and exit metadata.

## Workflow

1. Run `vcsm status`.
2. If the vault is locked, ask the user to run `vcsm unlock` in the trusted local prompt. Do not enter, request, pipe, store, or receive the password.
3. Run only the applicable name-based command from [references/handler-contract.md](references/handler-contract.md).
4. For execution, use only `vcsm run <project> <environment> <approved-action>`.
5. Report metadata and outcome only.

## Boundaries

- Never inspect the vault database, broker socket, process environment, clipboard, shell history, memory, or application files to find a value.
- Never run `env`, `printenv`, `set`, secret dump/export commands, or code intended to encode or reveal injected values.
- Never write values into arguments, files, logs, snapshots, screenshots, commits, issues, or chat.
- Never configure an action on the user's behalf. Prepare the proposed command and mappings, then let the user approve them through the trusted prompt.
- Never add a value-returning API, debug route, clipboard operation, `.env` writer, or raw password-manager adapter.
- Treat redacted process output as defense in depth, not permission to reveal or transform secrets.

If VCSM is unavailable or an operation would cross these boundaries, stop and explain the required trusted user action.
