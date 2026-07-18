# Protected-mode architecture and implementation status

## Runtime

VCSM runs natively as two small Go binaries:

1. `vcsm-broker` is a per-user background process that owns decrypted key material, vault access, generation, policy checks, and child-process injection.
2. `vcsm` is a thin client and trusted hidden master-password prompt. It contains no vault key after the request completes.

SQLite is embedded in the broker. No database service and no Docker daemon are required. Docker is supported only when a user approves `docker` as an action executable.

## Modules

| Module | Responsibility | Secret plaintext allowed? |
|---|---|---:|
| `internal/securecrypto` | Argon2id, HKDF, XChaCha20-Poly1305, zeroing | transiently |
| `internal/vault` | encrypted records, versions, action policy, audit chain | transiently |
| `internal/broker` | validates operations and orchestrates execution | transiently |
| `internal/generator` | compatibility-safe random values | until encrypted |
| `internal/ipc` | local framed request/response transport | unlock request only |
| `internal/runner` | starts the already-approved executable | child environment |
| `internal/redaction` | exact-value streaming redaction | transiently |
| `internal/power` | permits display sleep while blocking idle system sleep | never |

The existing `runner`, `redaction`, and `platform` packages are reused instead of introducing a second execution stack.

## Unlock and lifecycle

- Initial unlock requires a master password of 15–256 characters in a local terminal field.
- Argon2id defaults to 64 MiB, three iterations, and one lane. Its result derives the wrapping key; it is not used as the vault DEK.
- The DEK remains in broker memory until `vcsm lock`, broker restart, logout/reboot, or application/service restart.
- Screen lock does not stop new actions and does not terminate existing actions.
- The power module uses `caffeinate -i`, `systemd-inhibit`, or `SetThreadExecutionState`, depending on the OS. None requests the display to remain on.

## AI operations

AI-safe operations are status, list metadata, create, rotate, revoke, run approved action, and list audit metadata. There is no value-returning operation.

Administrative action configuration requires master-password reauthentication. The password is accepted only through the local permission-restricted endpoint, zeroed after use, and never returned or logged. The CLI refuses non-interactive input so an agent cannot place the password in a pipe, argument, or environment variable.

## Output policy

`metadata` is the default action output policy. Stdout and stderr are consumed and discarded; only action name and exit code return to the caller.

`redacted` is an explicit user-approved policy. It streams process output after removing exact current secret values, including values split across writes. It cannot reliably detect hashes, encodings, encryption, character-by-character output, files, or network exfiltration. Code receiving a secret must therefore be treated as trusted enough to possess it.

## Deliberately deferred

- Graphical UI and biometric wrapper provider.
- Password-manager API adapters. Password-manager auto-type into the hidden master-password field works today.
- Import from external password managers.
- Rotation hooks that update an external provider before activating the new local version.
- Packaging/signing/notarization and automatic binary updates.

These are separate adapters around the broker and vault; they do not require replacing the encrypted storage or AI-safe API.
