# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.1.x   | ✅        |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Send a report to: **security@orbit-sh.dev**

Include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Your suggested fix (optional)

We will acknowledge your report within **48 hours** and aim to release a fix within **7 days** for critical issues.

## Scope

In scope:

- Remote code execution via SSH or Docker API
- Authentication/authorization bypass
- Secrets exposure in logs or config files
- Path traversal vulnerabilities

Out of scope:

- Issues in dependencies (report upstream)
- Issues requiring physical access to the server
