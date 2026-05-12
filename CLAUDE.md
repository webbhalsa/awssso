# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o awssso .        # build binary
go build ./...            # verify all packages compile
go vet ./...              # lint
go test ./...             # run all tests
go test ./internal/awsconfig/...  # run a single package's tests
```

## Architecture

`awssso` is a CLI tool for AWS SSO login with interactive TUI profile selection.

**Command flow** — `cmd/` contains one file per subcommand wired into cobra via `cmd/root.go`:
- `setup` — prompts for SSO start URL/region, runs OIDC device auth, discovers all account+role combos via `sso:ListAccounts` + `sso:ListAccountRoles`, shows a multi-select checklist, writes `~/.aws/config`
- `login` — authenticates a session (OIDC device auth), then calls `pickProfileAndExport`
- `use` — skips session auth step, shows all profiles directly, re-authenticates transparently if token expired, then calls `pickProfileAndExport`
- `status` — reads config + cache, prints token validity per session and lists associated profiles
- `init` — prints a bash/zsh/fish shell function that wraps the binary so `awssso login` and `awssso use` can `eval` their output into the calling shell

**The eval pattern** — `login` and `use` print only `export AWS_PROFILE=<name>` to stdout. All other output (progress, errors) goes to stderr. The shell function from `awssso init` captures stdout and evals it, so the env var lands in the user's shell session. Users add `eval "$(awssso init)"` once to their `.zshrc`.

**`internal/awsconfig`** — hand-rolled INI parser for `~/.aws/config`. Understands both `[sso-session <name>]` blocks and `[profile <name>]` blocks. `Config.Write()` rewrites the whole file (does not preserve comments or ordering from the original).

**`internal/sso`** — three files:
- `oidc.go` — OIDC device authorization flow (`RegisterClient` → `StartDeviceAuthorization` → poll `CreateToken`) and `ListAccounts`/`ListAccountRoles` for discovery
- `cache.go` — reads/writes `~/.aws/sso/cache/*.json` token files; falls back to scanning all cache files by `startUrl` field when the SHA1-keyed filename doesn't exist
- `credentials.go` — `GetRoleCredentials` via the SSO API (fetches temporary AWS creds for a given account+role); currently unused but retained for future use

**`internal/tui`** — two bubbletea components:
- `list.go` — single-item picker with fuzzy filter (`bubbles/list`)
- `checklist.go` — multi-select with space-to-toggle, `a` for all/none, type-to-filter; used in `setup` for choosing which account/role combos to add as profiles

**Config writing** — `awsconfig.Config.Write()` regenerates the entire file from in-memory state. Existing config entries not touched by `awssso setup` are not preserved unless they were parsed and held in memory. Profiles are stored in a map (unordered); sessions are written before profiles.
