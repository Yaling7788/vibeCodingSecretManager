# Handler Contract

## Purpose

The handler is the only interface an AI coding agent may use for local secret operations. It must keep values hidden from stdout, stderr, logs, command arguments, files, and chat.

## Required Commands

```bash
./scripts/secret-handler list <project> <environment>
./scripts/secret-handler check <project> <environment>
./scripts/secret-handler create <project> <environment> <ENV_NAME>
./scripts/secret-handler rotate <project> <environment> <ENV_NAME>
./scripts/secret-handler delete <project> <environment> <ENV_NAME>
./scripts/secret-handler run <project> <environment> -- <command...>
```

## Output Rules

Allowed output:

```text
DATABASE_URL -> SampleWebApp/Dev/DATABASE_URL
Created DATABASE_URL
Rotated RESEND_API_KEY
Deleted OLD_TOKEN
Check passed
```

Forbidden output:

```text
DATABASE_URL=postgresql://...
old value: ...
new value: ...
secret preview: sk-...
```

## Create

The handler must:

1. Resolve `ENV_NAME` to an allowed vault entry path from config, or ask the human to approve adding a new mapping.
2. Prompt locally for the value with hidden input.
3. Store the value in the password manager.
4. Print only the environment variable name and entry path.

## Rotate

The handler must:

1. Confirm the target by `ENV_NAME` and entry path.
2. Prompt locally for the replacement value with hidden input, or generate a value locally without printing it.
3. Update the password-manager entry.
4. Print only status.

## Delete

The handler must:

1. Confirm the target by name and path.
2. Delete the password-manager entry or disable the manifest mapping, depending on user intent.
3. Print only status.

## Run

The handler must:

1. Load configured secrets.
2. Inject values only into the child process environment.
3. Forward child stdout and stderr.
4. Avoid printing the injected environment.

## Implementation Notes

- Do not pass secret values as command-line arguments.
- Do not write `.env` files by default.
- Prefer hidden terminal input for human-provided values.
- Redact any known secret value from handler errors before printing.
- Use development and test credentials, not production credentials.
