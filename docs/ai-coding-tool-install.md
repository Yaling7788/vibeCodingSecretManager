# AI Coding Tool Install

This page is meant to be pasted into Claude Code, Codex, Cursor, or another local coding tool when setting up an application repository.

## Paste-Ready Prompt

```text
Install vibeCodingSecretManager in this application repo using KeePassXC as the local secret manager.

Use this Git repository:
https://github.com/Yaling7788/vibeCodingSecretManager

Your job:
1. Install the CLI using the repository's scripts/install.sh workflow.
2. Create or update ./scripts/secret-dev.
3. Create or update .env.example with placeholders only.
4. Update .gitignore so .env files and KeePassXC files are ignored.
5. Create AI coding rules in CLAUDE.md or the equivalent local agent instruction file.
6. Configure ~/.config/vibeCodingSecretManager/config.yaml with project name, environment, project root, KeePassXC database path, and secret entry paths.
7. Use KeePassXC as the only secret value store.

Security rules:
- Do not ask me to paste real secret values into chat.
- Do not read, print, inspect, infer, retrieve, summarize, or copy real secret values.
- Do not run env, printenv, set, export, keepassxc-cli directly, cat .env*, rg .env*, clipboard reads, shell history reads, or cloud secret read commands.
- If a real value is needed, instruct me to enter it into KeePassXC or into a local hidden prompt.
- Report only variable names and KeePassXC entry paths.

After setup, run only:
./scripts/secret-dev list-secrets

Ask me to create the KeePassXC entries manually, then run:
./scripts/secret-dev check-secrets
```

## Single Command Install

Run this from the application repo you want to protect:

```bash
VCSM_REPO_URL=https://github.com/Yaling7788/vibeCodingSecretManager.git \
VCSM_PROJECT=sample-webapp \
VCSM_ENV=dev \
VCSM_SECRETS=DATABASE_URL,OPENAI_API_KEY \
sh -c "$(curl -fsSL https://raw.githubusercontent.com/Yaling7788/vibeCodingSecretManager/main/scripts/install.sh)"
```

Customize:

- `VCSM_PROJECT`: local project key used in config.
- `VCSM_ENV`: environment name, usually `dev`.
- `VCSM_SECRETS`: comma-separated environment variable names.
- `VCSM_DATABASE`: KeePassXC database path, defaults to `~/KeePass/example-dev.kdbx`.
- `VCSM_KEY_FILE`: optional KeePassXC key file path.
- `VCSM_CLI_PATH`: optional path to `keepassxc-cli`.

The installer creates:

- `./scripts/secret-dev`
- `.env.example`
- `.gitignore` secret ignores
- `CLAUDE.md` if missing
- `~/.config/vibeCodingSecretManager/config.yaml` if missing

## Master Password Policy

The human should type the KeePassXC master password into the local hidden prompt when running `check-secrets` or commands that inject secrets.

Do not let an AI coding agent manage, store, paste, or receive the KeePassXC master password.

Headless mode where the AI can access the master key is not safe for this security model. If the AI can unlock the vault without the human, it can retrieve every configured secret and any other entry reachable with that key. At that point, KeePassXC no longer protects secrets from the AI; it only moves the exposure from `.env` files to the AI-controlled unlock path.

Acceptable automation:

- Human unlocks KeePassXC through a hidden prompt.
- Runner injects only configured secrets into one child process.
- AI can run approved wrapper commands but cannot see the master password or secret values.

Risky automation:

- Master password in environment variables.
- Master password in files, shell history, CI logs, keychain items accessible to the agent, or command arguments.
- Long-lived unlocked headless session controlled by the AI.
- Any "show", "export", or debug command that reveals values.

## After Install

Create the KeePassXC entries named in the generated config. Store each value in the KeePassXC entry Password field.

Then run:

```bash
./scripts/secret-dev list-secrets
./scripts/secret-dev check-secrets
./scripts/secret-dev up
```
