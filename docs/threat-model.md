# Threat Model

## Protects Against

1. Accidental `.env` reads by AI coding agents.
2. Accidental commits of repo-local `.env` files.
3. AI agents inspecting project-local secret files while debugging.
4. Secrets being stored in source-controlled files.
5. Casual terminal output leaks from the secret retrieval step.

## Does Not Fully Protect Against

1. Malicious code intentionally printing environment variables.
2. AI agents running unapproved commands with full shell access.
3. A compromised local machine.
4. A compromised KeePassXC database, key file, or master password.
5. Secrets leaked by application logs.
6. Secrets visible to process inspection tools available to the same user account.

## Required User Discipline

1. Keep KeePassXC databases outside the repo.
2. Keep KeePassXC key files outside the repo.
3. Do not approve forbidden shell commands.
4. Use dev and test credentials, not production credentials.
5. Rotate secrets if exposed.
6. Run a secret scanner such as gitleaks before committing.

## Design Choices

`vibeCodingSecretManager` does not start a local HTTP API. There is no localhost endpoint for secrets.

The config maps specific environment variable names to specific KeePassXC entries. Agents cannot ask the runner for arbitrary entries unless the user explicitly adds those entries to the config.

Secrets are passed only through the child process environment. The runner does not write `.env` files by default.
