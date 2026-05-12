# awssso

A CLI for AWS SSO — discover your accounts and roles, configure `~/.aws/config`, and authenticate with an interactive profile picker.

## Installation

```bash
brew tap webbhalsa/tap
brew install awssso
```

Shell integration is added to your `.zshrc`, `.bashrc`, or `config.fish` automatically during installation. If it wasn't (e.g. you use a non-standard shell), add it manually:

```bash
# zsh / bash
eval "$(awssso init)"

# fish
awssso init | source
```

To upgrade to the latest version:

```bash
brew upgrade awssso
```

## Usage

### Setup

Run once to discover your AWS accounts and roles and generate `~/.aws/config`:

```bash
awssso setup
```

This will:
1. Prompt for your SSO start URL and region
2. Open a browser for authentication
3. Discover all accounts and roles you have access to
4. Show an interactive checklist — pick which combinations to add as profiles
5. Write the profiles to `~/.aws/config` using the modern `[sso-session]` format

### Login

Authenticate and select which profile to use in the current shell:

```bash
awssso login
```

If you have multiple SSO sessions configured, you'll be asked to pick one first. After authenticating, a profile picker lets you choose which account/role to activate. The selected profile is exported as `AWS_PROFILE` into your shell session.

### Use

Pick any profile without re-authenticating (re-authenticates transparently if the token is expired):

```bash
awssso use
```

### Status

Show token validity for all configured SSO sessions and their associated profiles:

```bash
awssso status
```

### Update notifications

If a newer version is available, a banner is printed to stderr after any command:

```
Update available: v1.0.0 → v1.1.0  Run: brew upgrade awssso
```

---

## How shell integration works

`awssso login` and `awssso use` print `export AWS_PROFILE=<name>` to stdout. The shell function installed by `awssso init` captures this and evals it, so the variable lands in your current shell session — no subshell, no manual export.

All other output (progress messages, errors) goes to stderr and is shown in the terminal but never evaluated.

---

## Releasing a new version

1. Make sure your changes are merged to `main`.
2. Commit and tag:
   ```bash
   git commit -m "release v1.0.0"
   git tag v1.0.0
   git push origin main --tags
   ```
3. The [Release workflow](https://github.com/jesperblomquist/awssso/actions/workflows/release.yml) will automatically build binaries for macOS and Linux and publish a GitHub Release. The Homebrew formula in [webbhalsa/homebrew-tap](https://github.com/webbhalsa/homebrew-tap) will be updated automatically.

Tags must follow [semver](https://semver.org/) and start with `v` (e.g. `v1.0.0`).

You'll need a `HOMEBREW_TAP_GITHUB_TOKEN` secret set in the repository with write access to the tap repo.
