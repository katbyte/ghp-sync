## v0.1.0 (2026-08-03)

First tagged release of ghp-sync, a small utility to sync GitHub issues and PRs to GitHub Projects.

- sync issues and PRs from one or more repos to a GitHub Project (`issues`, `prs`) and between two projects (`project`)
- add `--sync-linked-issue-fields` to copy project field values from a PR's linked issue
- release binaries with goreleaser (linux/darwin/windows/freebsd/openbsd/solaris) on tagged releases
- publish a homebrew formula to `katbyte/homebrew-tap` on release
- modernize tooling: go 1.25, golangci-lint v2 with expanded linters, gofumpt, CI workflows (build/test/lint/depscheck/govulncheck/codeql), dependabot
- fix version reporting so `ghp-sync version` shows the real version and commit
