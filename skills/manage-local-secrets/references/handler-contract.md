# VCSM command contract

## Agent-safe commands

```bash
vcsm status
vcsm secret create <project> <environment> <name> [--profile default|alnum|password|database]
vcsm secret rotate <project> <environment> <name> [--profile default|alnum|password|database]
vcsm secret revoke <project> <environment> <name>
vcsm secret list <project> <environment>
vcsm action list <project> <environment>
vcsm run <project> <environment> <approved-action>
vcsm audit [limit]
```

These commands may return names, identifiers, generation profile, entropy, version, state, timestamps, action metadata, audit metadata, and exit code. They must never return a secret value.

## Trusted user commands

Do not execute or automate commands that request the master password. Show the exact command for the user to run locally:

```bash
vcsm init
vcsm unlock
vcsm action configure <project> <environment> <name> [--cwd dir] [--secret ENV=secret-name] [--output metadata|redacted] -- executable [args...]
```

Use `metadata` output unless the user explicitly approves `redacted`. Never approve arbitrary actions for the user.

## Failure handling

- Locked: ask the user to run `vcsm unlock`, then retry the original name-based operation.
- Missing secret: report its name and scope only.
- Missing action: propose the action metadata and ask the user to approve it.
- Broker unavailable: report `vcsm status` failure and recommend starting the per-user broker service.
- Command failure: report the action name and exit code; do not inspect the injected environment.
