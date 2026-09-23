## v0.5.0 (2026-09-23)

- add `Last Reviewer` PR field, who left the most recent submitted review; refreshed on open PRs too ([#27](https://github.com/katbyte/ghp-sync/pull/27))

## v0.4.0 (2026-09-23)

- add `Reviewed At` PR field, the date of the most recent submitted review; refreshed on open PRs too ([#25](https://github.com/katbyte/ghp-sync/pull/25))
- ci: scan the release binary, the published docker image and a build of main with trivy for known vulnerabilities; a release reruns the failed scan in place ([#26](https://github.com/katbyte/ghp-sync/pull/26))
- docker: upgrade alpine packages when building the image, so a release picks up base image security fixes ([#26](https://github.com/katbyte/ghp-sync/pull/26))
- bump golang.org/x/text to v0.42.0 for CVE-2026-56852, a hang on invalid UTF-8 input; ghp-sync never calls the affected code, but the binary now scans clean ([#26](https://github.com/katbyte/ghp-sync/pull/26))

## v0.3.2 (2026-09-15)

- publish the `v`-prefixed image tag too, so `:v0.3.2` works alongside `:0.3.2`, `:0.3` and `:latest` ([4119fc7](https://github.com/katbyte/ghp-sync/commit/4119fc7))

## v0.3.1 (2026-09-15)

- publish a multi-arch image to `ghcr.io/katbyte/ghp-sync` on release, signed with cosign ([25e7e79](https://github.com/katbyte/ghp-sync/commit/25e7e79))
- fix the docker build failing on go 1.25; the go version now comes from `.go-version` ([25e7e79](https://github.com/katbyte/ghp-sync/commit/25e7e79))
- build the image from a prebuilt binary on alpine, ~110MB instead of ~1GB ([25e7e79](https://github.com/katbyte/ghp-sync/commit/25e7e79))
- compose pulls the published image instead of building locally ([25e7e79](https://github.com/katbyte/ghp-sync/commit/25e7e79))
- add `.dockerignore` ([25e7e79](https://github.com/katbyte/ghp-sync/commit/25e7e79))

## v0.3.0 (2026-09-15)

- fix env var lists splitting on spaces, so `GITHUB_PR_POPULATE_FIELDS=PR#,Approved By` works ([#17](https://github.com/katbyte/ghp-sync/pull/17))
- pin dev tools in `.tools/`, built by `make tools` ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- add actionlint, yamllint, shellcheck, and typos checks ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- enable revive and azproviderlint linters ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- pin github actions by commit hash, workflow tokens read-only ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- bump codeql-action to v4.38.0 and group init/analyze for dependabot ([#23](https://github.com/katbyte/ghp-sync/pull/23))
- sign releases with cosign, attach build provenance ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- add `SECURITY.md` and README badges ([#18](https://github.com/katbyte/ghp-sync/pull/18))
- move clog and version to go-kt ([60e0d33](https://github.com/katbyte/ghp-sync/commit/60e0d33), [9d0f4a0](https://github.com/katbyte/ghp-sync/commit/9d0f4a0))

## v0.2.0 (2026-08-27)

- add `--merged-by` and `--merged-since` filters for syncing merged PRs; `--merged-since` walks PRs by update time so it can stop early instead of crawling full repo history ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- add `--filters-only` to disable auto-including PRs already in the project ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- add `prs refresh` subcommand to update closed/merged PR items already on the board, with `--include-open` to also refresh a safe subset of fields on open PRs ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95), [59c36b9](https://github.com/katbyte/ghp-sync/commit/59c36b9))
- add new PR fields: `Merged By`, `Merged At`, `Reviewed By`, `Approved By` ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95), [59c36b9](https://github.com/katbyte/ghp-sync/commit/59c36b9))
- add `Changes Requested By` PR field: reviewers who requested changes in order of first request, each with their request count and total comments across those requests, e.g. `sreallymatt(×2 ✎10), katbyte(×1 ✎1)` ([9bbc1b1](https://github.com/katbyte/ghp-sync/commit/9bbc1b1))
- `prs refresh` now diffs against current board values: only changed fields are updated, output shows `field: old -> new`, and up-to-date items are skipped ([9bbc1b1](https://github.com/katbyte/ghp-sync/commit/9bbc1b1))
- add `CI` (`✅` passing, `❌` failing, `🕒` running) and `Mergeable` (`✅` clean, `❌` merge conflicts, `?` unknown) PR fields; both cleared when a PR closes, and `prs refresh --include-open` fetches them live for open PRs. When github hasn't computed mergeability yet the pr is re-queried with retries (as in tctest) so `?` is only stamped when it never settles ([9bbc1b1](https://github.com/katbyte/ghp-sync/commit/9bbc1b1))
- make author/assignee/merged-by/reviewer login matching case-insensitive ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- missing project fields now warn and skip by default instead of erroring; add `--strict` to error instead ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- dry run now prints each field value, and sync output shows why each PR matched the filters plus a link to the PR ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- retry transient network errors (http2 stream resets, connection resets, timeouts) in both GraphQL paths ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- fix project field loading: handle iteration fields and fetch up to 100 fields (was 40, silently truncating) ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))
- fix `GetItems` GraphQL query using `singleSelectOptionId` instead of `optionId` ([59c36b9](https://github.com/katbyte/ghp-sync/commit/59c36b9))
- `SYNC_CMD` in the docker image now supports subcommand arguments (e.g. `prs refresh`) ([4a12d95](https://github.com/katbyte/ghp-sync/commit/4a12d95))

## v0.1.0 (2026-08-03)

First tagged release of ghp-sync, a small utility to sync GitHub issues and PRs to GitHub Projects.

- sync issues and PRs from one or more repos to a GitHub Project (`issues`, `prs`) and between two projects (`project`)
- add `--sync-linked-issue-fields` to copy project field values from a PR's linked issue ([f29dcc4](https://github.com/katbyte/ghp-sync/commit/f29dcc4))
- release binaries with goreleaser (linux/darwin/windows/freebsd/openbsd/solaris) on tagged releases ([586f97f](https://github.com/katbyte/ghp-sync/commit/586f97f))
- publish a homebrew formula to `katbyte/homebrew-tap` on release ([586f97f](https://github.com/katbyte/ghp-sync/commit/586f97f))
- modernize tooling: go 1.25, golangci-lint v2 with expanded linters, gofumpt, CI workflows (build/test/lint/depscheck/govulncheck/codeql), dependabot ([586f97f](https://github.com/katbyte/ghp-sync/commit/586f97f), [#5](https://github.com/katbyte/ghp-sync/pull/5))
- fix version reporting so `ghp-sync version` shows the real version and commit ([586f97f](https://github.com/katbyte/ghp-sync/commit/586f97f))
