## v0.4.0 (2026-09-23)

- add `Last Reviewed At` PR field, the date of the most recent submitted review; refreshed on open PRs too
- ci: scan the release binary, the published docker image and a build of main with trivy for known vulnerabilities; a release reruns the failed scan in place
- docker: upgrade alpine packages when building the image, so a release picks up base image security fixes
- bump golang.org/x/text to v0.42.0 for CVE-2026-56852, a hang on invalid UTF-8 input; ghp-sync never calls the affected code, but the binary now scans clean

## v0.3.2 (2026-09-15)

- publish the `v`-prefixed image tag too, so `:v0.3.2` works alongside `:0.3.2`, `:0.3` and `:latest`

## v0.3.1 (2026-09-15)

- publish a multi-arch image to `ghcr.io/katbyte/ghp-sync` on release, signed with cosign
- fix the docker build failing on go 1.25; the go version now comes from `.go-version`
- build the image from a prebuilt binary on alpine, ~110MB instead of ~1GB
- compose pulls the published image instead of building locally
- add `.dockerignore`

## v0.3.0 (2026-09-15)

- fix env var lists splitting on spaces, so `GITHUB_PR_POPULATE_FIELDS=PR#,Approved By` works
- pin dev tools in `.tools/`, built by `make tools`
- add actionlint, yamllint, shellcheck, and typos checks
- enable revive and azproviderlint linters
- pin github actions by commit hash, workflow tokens read-only
- bump codeql-action to v4.38.0 and group init/analyze for dependabot
- sign releases with cosign, attach build provenance
- add `SECURITY.md` and README badges
- move clog and version to go-kt

## v0.2.0 (2026-08-27)

- add `--merged-by` and `--merged-since` filters for syncing merged PRs; `--merged-since` walks PRs by update time so it can stop early instead of crawling full repo history
- add `--filters-only` to disable auto-including PRs already in the project
- add `prs refresh` subcommand to update closed/merged PR items already on the board, with `--include-open` to also refresh a safe subset of fields on open PRs
- add new PR fields: `Merged By`, `Merged At`, `Reviewed By`, `Approved By`
- add `Changes Requested By` PR field: reviewers who requested changes in order of first request, each with their request count and total comments across those requests, e.g. `sreallymatt(×2 ✎10), katbyte(×1 ✎1)`
- `prs refresh` now diffs against current board values: only changed fields are updated, output shows `field: old -> new`, and up-to-date items are skipped
- add `CI` (`✅` passing, `❌` failing, `🕒` running) and `Mergeable` (`✅` clean, `❌` merge conflicts, `?` unknown) PR fields; both cleared when a PR closes, and `prs refresh --include-open` fetches them live for open PRs. When github hasn't computed mergeability yet the pr is re-queried with retries (as in tctest) so `?` is only stamped when it never settles
- make author/assignee/merged-by/reviewer login matching case-insensitive
- missing project fields now warn and skip by default instead of erroring; add `--strict` to error instead
- dry run now prints each field value, and sync output shows why each PR matched the filters plus a link to the PR
- retry transient network errors (http2 stream resets, connection resets, timeouts) in both GraphQL paths
- fix project field loading: handle iteration fields and fetch up to 100 fields (was 40, silently truncating)
- fix `GetItems` GraphQL query using `singleSelectOptionId` instead of `optionId`
- `SYNC_CMD` in the docker image now supports subcommand arguments (e.g. `prs refresh`)

## v0.1.0 (2026-08-03)

First tagged release of ghp-sync, a small utility to sync GitHub issues and PRs to GitHub Projects.

- sync issues and PRs from one or more repos to a GitHub Project (`issues`, `prs`) and between two projects (`project`)
- add `--sync-linked-issue-fields` to copy project field values from a PR's linked issue
- release binaries with goreleaser (linux/darwin/windows/freebsd/openbsd/solaris) on tagged releases
- publish a homebrew formula to `katbyte/homebrew-tap` on release
- modernize tooling: go 1.25, golangci-lint v2 with expanded linters, gofumpt, CI workflows (build/test/lint/depscheck/govulncheck/codeql), dependabot
- fix version reporting so `ghp-sync version` shows the real version and commit
