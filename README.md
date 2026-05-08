# vibeCodingSecretManager

`vibeCodingSecretManager` is a small local CLI for safely injecting development secrets into application processes without storing `.env` files in your repository.

It is designed for developers using AI coding agents such as Claude Code, Codex, Cursor, or other local assistants that can read and edit project files.

Secrets stay in KeePassXC. The AI agent runs a safe wrapper command. The wrapper injects secrets into the child process environment without printing or writing them to disk.

## Install

From an application repo, use the one-command bootstrap:

```bash
VCSM_REPO_URL=https://github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager.git \
VCSM_PROJECT=sample-webapp \
VCSM_ENV=dev \
VCSM_SECRETS=DATABASE_URL,OPENAI_API_KEY \
sh -c "$(curl -fsSL https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/main/scripts/install.sh)"
```

Or install only the CLI:

```bash
go install github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/cmd/vibeCodingSecretManager@latest
```

For local development:

```bash
go build ./cmd/vibeCodingSecretManager
```

## Quick Start

For AI coding tools, start with [AI coding tool install](docs/ai-coding-tool-install.md). It includes a paste-ready prompt and a one-command setup flow.

Create a starter config:

```bash
vibeCodingSecretManager init
```

Edit:

```text
~/.config/vibeCodingSecretManager/config.yaml
```

Example:

```yaml
vault:
  type: keepassxc
  database: ~/KeePass/example-dev.kdbx
  key_file: ~/KeePass/example-dev.key
  cli_path: auto

projects:
  sample-webapp:
    root: ~/Projects/sample-webapp
    environments:
      dev:
        secrets:
          DATABASE_URL: SampleWebApp/Dev/DATABASE_URL
          RESEND_API_KEY: SampleWebApp/Dev/RESEND_API_KEY
          GOOGLE_CLIENT_ID: SampleWebApp/Dev/GOOGLE_CLIENT_ID
          GOOGLE_CLIENT_SECRET: SampleWebApp/Dev/GOOGLE_CLIENT_SECRET
```

List configured variables without values:

```bash
vibeCodingSecretManager list sample-webapp dev
```

Check the configuration and KeePassXC entries:

```bash
vibeCodingSecretManager check sample-webapp dev
```

Run a development command:

```bash
vibeCodingSecretManager run sample-webapp dev -- npm run dev
```

Docker Compose:

```bash
vibeCodingSecretManager run sample-webapp dev -- docker compose up
```

## Security Model

Core principle:

```text
The AI coding agent may run safe project commands.
The AI coding agent must not read, print, copy, or inspect secrets.
```

`vibeCodingSecretManager` retrieves only the secrets configured for a specific project and environment. It injects them into the child process environment and never prints the values.

Use development and test credentials. This tool is a guardrail for local workflow safety, not a sandbox for malicious code.

The KeePassXC master password must be entered by the human into a local hidden prompt. Do not store it in files, environment variables, shell history, command arguments, or any AI-readable key store.

Headless mode where the AI can unlock KeePassXC is not safe for this threat model. If the AI controls the master key, it can unlock and retrieve the vault contents. Use a human unlock step plus approved wrapper commands instead.

## KeePassXC Naming

Recommended entry path format:

```text
<Project>/<Environment>/<VARIABLE_NAME>
```

Examples:

```text
SampleWebApp/Dev/DATABASE_URL
SampleWebApp/Dev/RESEND_API_KEY
SampleAPI/Dev/AWS_ACCESS_KEY_ID
```

Store the actual secret in each entry's Password field.

## Commands

```bash
vibeCodingSecretManager run <project> <environment> -- <command...>
vibeCodingSecretManager check <project> <environment>
vibeCodingSecretManager list <project> <environment>
vibeCodingSecretManager init
```

Use `--config path` before the command to load a non-default config file.

## AI Agent Wrapper

Inside an application repo, create a wrapper such as:

```bash
#!/bin/bash
set -euo pipefail

COMMAND="${1:-}"

case "$COMMAND" in
  up)
    exec vibeCodingSecretManager run sample-webapp dev -- npm run dev
    ;;
  docker)
    exec vibeCodingSecretManager run sample-webapp dev -- docker compose up
    ;;
  build)
    exec vibeCodingSecretManager run sample-webapp dev -- npm run build
    ;;
  test)
    exec npm test
    ;;
  lint)
    exec npm run lint
    ;;
  check-secrets)
    exec vibeCodingSecretManager check sample-webapp dev
    ;;
  *)
    echo "Usage: ./scripts/claude-dev {up|docker|build|test|lint|check-secrets}"
    exit 1
    ;;
esac
```

Then allow the agent to run:

```bash
./scripts/claude-dev up
```

Do not let the agent run `keepassxc-cli`, `printenv`, `env`, or commands that read `.env` files.

## Documentation

- [Threat model](docs/threat-model.md)
- [KeePassXC setup](docs/keepassxc-setup.md)
- [AI coding tool install](docs/ai-coding-tool-install.md)
- [Claude Code setup](docs/claude-code-setup.md)
- [Examples](docs/examples.md)
