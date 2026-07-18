# Security Policy

## Supported Versions

This project is currently pre-1.0. Security fixes will target the latest release.

## Reporting a Vulnerability

Please do not open public issues for vulnerabilities that could leak secrets.

Email the maintainer or use GitHub private vulnerability reporting when enabled. Include:

- A short description of the issue
- Steps to reproduce
- Affected version or commit
- Whether secret values can be printed, written, or exposed to child commands unexpectedly

## Security Boundaries

VCSM is a local workflow guardrail. It does not sandbox arbitrary commands, prevent malicious application code from reading its own environment, or protect a compromised machine.

Use development credentials and rotate any secret that may have been exposed.
