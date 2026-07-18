# Threat model

Protected mode is designed to keep secret values out of source trees, AI conversations, command arguments, configuration files, normal IPC responses, and audit records.

## Protected cases

- Accidental AI reads of `.env` or password-manager databases.
- Accidental source-control commits of development secrets.
- AI requests to retrieve or export a value: no such broker operation exists.
- AI selection of arbitrary commands: execution requires a previously user-approved action name.
- Offline theft of the vault database without the master password.
- Simple output of an exact secret when the user explicitly enables redacted output.

## Out of scope

- Malware, administrator/root access, process debugging, or a compromised user session.
- Malicious code that receives a secret and encodes, writes, or transmits it.
- A dependency or approved executable that exfiltrates its environment.
- A weak or captured master password.
- Production-secret governance, remote authorization, multi-user sharing, or host attestation.

An AI agent with unrestricted execution as the same OS user can modify code run by an approved action and can often inspect peer processes. VCSM reduces accidental disclosure and constrains normal operations; it is not a sandbox. Use scoped development credentials and rotate exposed values.

## Defaults

- Child stdout/stderr is discarded and only exit metadata returns.
- Unlock persists until manual lock or broker restart; screen lock does not pause jobs.
- The display may sleep, but idle system sleep is inhibited while unlocked.
- The master password is entered through a local interactive hidden prompt, never an argument, environment variable, or file.
- Secret generation uses compatibility-safe alphabets with at least 200 bits of entropy.
