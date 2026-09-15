# Security Policy

## Supported Versions

Only the [latest release](https://github.com/katbyte/ghp-sync/releases/latest) is supported — please update before reporting an issue.

## Reporting a Vulnerability

Please **do not** open a public issue for security vulnerabilities.

Instead, report privately via [GitHub's private vulnerability reporting](https://github.com/katbyte/ghp-sync/security/advisories/new).

I will do my best to acknowledge reports within 2 weeks and aim to release a fix or mitigation within 6 weeks for confirmed issues; timelines are best-effort.

## Scope

`ghp-sync` handles GitHub API tokens supplied via flags or environment variables and writes to GitHub Project boards. Issues involving token leakage (e.g. tokens appearing in logs, output, or error messages) are particularly relevant.
