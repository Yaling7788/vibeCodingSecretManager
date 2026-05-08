# Examples

## Docker Compose

```yaml
services:
  api:
    build:
      context: .
    environment:
      DATABASE_URL: ${DATABASE_URL}
      RESEND_API_KEY: ${RESEND_API_KEY}
      GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID}
      GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET}
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    ports:
      - "3000:3000"
```

Run:

```bash
vibeCodingSecretManager run sample-webapp dev -- docker compose up
```

Do not use:

```yaml
env_file:
  - .env
```

## `.env.example`

```env
DATABASE_URL=replace_me
RESEND_API_KEY=replace_me
GOOGLE_CLIENT_ID=replace_me
GOOGLE_CLIENT_SECRET=replace_me
ANTHROPIC_API_KEY=replace_me
OPENAI_API_KEY=replace_me
```

## `.gitignore`

```gitignore
# Secret files
.env
.env.*
!.env.example

*.pem
*.key
*.p12
*.pfx

secrets/
Secrets/

# KeePassXC
*.kdbx
*.kdbx.lock
```
