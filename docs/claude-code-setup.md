# Claude Code Setup

Add a wrapper script to the application repo:

```text
scripts/claude-dev
```

Example:

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
    echo "Usage:"
    echo "  ./scripts/claude-dev up"
    echo "  ./scripts/claude-dev docker"
    echo "  ./scripts/claude-dev build"
    echo "  ./scripts/claude-dev test"
    echo "  ./scripts/claude-dev lint"
    echo "  ./scripts/claude-dev check-secrets"
    exit 1
    ;;
esac
```

Make it executable:

```bash
chmod +x scripts/claude-dev
```

## CLAUDE.md Rules

Add this to `CLAUDE.md`:

````markdown
# Claude Code Security Rules

## Secret Access

You must not read, print, retrieve, inspect, summarize, modify, or infer real secrets.

Secrets are managed by KeePassXC and injected through `vibeCodingSecretManager`.

## Allowed Commands

You may run:

```bash
./scripts/claude-dev up
./scripts/claude-dev docker
./scripts/claude-dev build
./scripts/claude-dev test
./scripts/claude-dev lint
./scripts/claude-dev check-secrets
```

## Forbidden Commands

Do not run:

```bash
cat .env*
less .env*
more .env*
grep .env*
rg .env*
printenv
env
set
export
keepassxc-cli
security find-generic-password
aws secretsmanager get-secret-value
aws ssm get-parameter --with-decryption
curl
wget
nc
scp
rsync
```

## Forbidden Files

Do not read:

```text
.env
.env.local
.env.production
.env.*
~/Secrets/**
~/.aws/**
~/.ssh/**
KeePassXC database files
KeePassXC key files
```

## Coding Rule

Use `.env.example` only for variable names and placeholders.

Never hardcode real secrets.

Never log environment variable values.

Never print `process.env`.

If the application needs real secrets, run only the approved wrapper:

```bash
./scripts/claude-dev up
```
````

## `.claude/settings.local.json`

```json
{
  "permissions": {
    "allow": [
      "Bash(./scripts/claude-dev up)",
      "Bash(./scripts/claude-dev docker)",
      "Bash(./scripts/claude-dev build)",
      "Bash(./scripts/claude-dev test)",
      "Bash(./scripts/claude-dev lint)",
      "Bash(./scripts/claude-dev check-secrets)"
    ],
    "deny": [
      "Read(./.env)",
      "Read(./.env.local)",
      "Read(./.env.production)",
      "Read(./.env.*)",
      "Read(./**/.env)",
      "Read(./**/.env.*)",
      "Read(~/Secrets/**)",
      "Read(~/.aws/**)",
      "Read(~/.ssh/**)",
      "Bash(cat .env*)",
      "Bash(less .env*)",
      "Bash(more .env*)",
      "Bash(grep * .env*)",
      "Bash(rg * .env*)",
      "Bash(printenv*)",
      "Bash(env)",
      "Bash(set)",
      "Bash(export)",
      "Bash(keepassxc-cli *)",
      "Bash(security find-generic-password*)",
      "Bash(aws secretsmanager get-secret-value*)",
      "Bash(aws ssm get-parameter*)",
      "Bash(curl *)",
      "Bash(wget *)",
      "Bash(nc *)",
      "Bash(scp *)",
      "Bash(rsync *)"
    ],
    "ask": [
      "Bash(*)"
    ]
  }
}
```
