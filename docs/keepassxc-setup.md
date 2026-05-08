# KeePassXC Setup

## Install KeePassXC CLI

Install KeePassXC from the official package for your platform. Confirm the CLI is available:

```bash
keepassxc-cli --version
```

If it is not on `PATH`, set `vault.cli_path` in your config.

## Create a Database

Recommended local paths:

```text
~/KeePass/example-dev.kdbx
~/KeePass/example-dev.key
```

Do not place these files inside any source repository.

## Entry Naming

Use:

```text
<Project>/<Environment>/<VARIABLE_NAME>
```

Examples:

```text
SampleWebApp/Dev/DATABASE_URL
SampleWebApp/Dev/RESEND_API_KEY
SampleWebApp/Dev/GOOGLE_CLIENT_ID
SampleWebApp/Dev/GOOGLE_CLIENT_SECRET
SampleWebApp/Dev/ANTHROPIC_API_KEY
SampleWebApp/Dev/OPENAI_API_KEY
```

Store each actual secret in the KeePassXC entry's Password field.

## Check Entries

After creating the config, run:

```bash
vibeCodingSecretManager check sample-webapp dev
```

The command prompts for your KeePassXC master password and verifies each configured entry without printing secret values.
