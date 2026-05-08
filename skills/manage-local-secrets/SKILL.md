---
name: manage-local-secrets
description: Operate local development secrets safely through approved secret-manager scripts or handlers. Use when asked to create, rotate, delete, list, check, audit, or retrieve secrets for local apps, KeePassXC, vibeCodingSecretManager, Claude Code, Codex, Cursor, or other AI coding workflows without exposing secret values.
---

# Manage Local Secrets

## Core Rule

Treat secret values as non-observable. Do not read, print, summarize, infer, copy, log, diff, commit, or paste real secret values.

Only use approved local handlers that are designed to hide values, prompt the human when needed, and return status without exposing the secret.

The human must enter the KeePassXC master password into a local hidden prompt. Do not manage, store, request, paste, or receive the master password. Do not create headless unlock flows where the AI controls the master key.

## Allowed Agent Work

- Create and edit secret manifests, templates, `.env.example`, wrapper scripts, and documentation.
- Add, remove, or rename configured secret references by environment variable name and vault entry path.
- Run safe commands that list names only, check existence, inject secrets into a child process, or invoke interactive human entry.
- Implement or improve handlers so they prompt the human for real values without echoing them.
- Report operation status by secret name and entry path only.

## Forbidden Agent Work

Never run commands whose purpose is to expose values, including `env`, `printenv`, `set`, `export`, `keepassxc-cli show`, `cat .env*`, `rg .env*`, password-manager raw reads, shell history reads, clipboard reads, or cloud secret-manager raw get commands.

Never write real values into repo files, logs, command arguments, test snapshots, screenshots, comments, Git commits, issue text, or chat output.

Never store the KeePassXC master password in environment variables, files, shell history, command arguments, keychain entries accessible to the agent, CI settings, or agent memory.

Never implement a localhost secret API, debug endpoint, value dump, temporary `.env` writer, or "show secret" command unless the user explicitly asks for a human-only emergency export. If that happens, pause and warn that this violates the normal boundary.

## Workflow

1. Identify the approved handler.
   - Prefer project scripts such as `./scripts/secret-handler`, `./scripts/claude-dev`, or the installed `vibeCodingSecretManager` CLI.
   - If no handler exists, create one that follows [handler-contract.md](references/handler-contract.md).

2. Identify the manifest.
   - Prefer `~/.config/vibeCodingSecretManager/config.yaml` or a project-local example config with placeholders.
   - Keep real database paths outside source repos unless the path itself is intentionally non-secret.
   - Keep `.env.example` placeholders only.

3. Plan the requested operation by names only.
   - Use environment variable names such as `DATABASE_URL`.
   - Use vault entry paths such as `SampleWebApp/Dev/DATABASE_URL`.
   - Do not ask the human to paste real values into chat.

4. Execute through the handler.
   - For create or rotate, invoke an interactive command that prompts the human locally with hidden input.
   - For delete, require explicit confirmation and delete by configured entry path.
   - For retrieve, prefer `run` injection into a child process. Do not display the value.
   - For audit/check, verify existence and configuration without values.

5. Summarize safely.
   - Say which secret names were created, rotated, deleted, checked, or injected.
   - Include no values and no value-derived hints.

## Command Contract

Use these verbs when the project handler supports them:

```bash
./scripts/secret-handler list <project> <environment>
./scripts/secret-handler check <project> <environment>
./scripts/secret-handler create <project> <environment> <ENV_NAME>
./scripts/secret-handler rotate <project> <environment> <ENV_NAME>
./scripts/secret-handler delete <project> <environment> <ENV_NAME>
./scripts/secret-handler run <project> <environment> -- <command...>
```

For the current CLI MVP, use:

```bash
vibeCodingSecretManager list <project> <environment>
vibeCodingSecretManager check <project> <environment>
vibeCodingSecretManager run <project> <environment> -- <command...>
```

If create, rotate, or delete are requested but missing, implement them in the handler or CLI with hidden prompts and no value output.

## Operation Notes

- **Create**: add or confirm the manifest mapping, then invoke a hidden local prompt to store the value in the password manager.
- **Rotate**: generate or prompt for a new value locally, update the password-manager entry, then optionally run `check`.
- **Delete**: confirm the entry path and environment variable name, then remove only that configured entry.
- **Retrieve**: inject into a child process with `run`; do not return or display the value.
- **Audit**: list names and paths, verify entries exist, scan repo for forbidden `.env` files and obvious leaked placeholders.

## If Blocked

If an operation requires a real value, ask the human to enter it into the local hidden prompt or password-manager UI. Do not ask them to paste it into chat.

If the user asks for headless AI-managed KeePassXC unlock, explain that this exposes the vault to the AI and violates the skill's security model. Offer human unlock plus approved wrapper commands instead.

If the handler would expose values, stop and modify the handler first.
