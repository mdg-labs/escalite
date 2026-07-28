# Contributing to Escalite

Thank you for contributing to Escalite Community Edition (AGPL-3.0). This document
covers the Developer Certificate of Origin (DCO), commit message conventions, and the
pull request workflow.

## Developer Certificate of Origin (DCO)

We use the [Developer Certificate of Origin](https://developercertificate.org/) (DCO)
instead of a Contributor License Agreement. Every commit in a pull request must include a
`Signed-off-by` trailer that certifies you have the right to submit the work under the
project license.

### DCO sign-off

Add a sign-off line to each commit message:

```text
Signed-off-by: Jane Doe <jane@example.com>
```

The easiest way is Git's `-s` / `--signoff` flag:

```bash
git commit -s -m "feat(web): add incident list page"
```

To sign off the most recent commit:

```bash
git commit --amend -s --no-edit
```

To sign off every commit on your branch (for example after rebasing):

```bash
git rebase --signoff origin/dev
```

CI rejects pull requests whose commits are missing a valid `Signed-off-by` line. The
sign-off name and email must match the commit author (case-insensitive for the email
address).

## SSH commit signing

Escalite also uses **SSH cryptographic signing** so commits show as **Verified** on
GitHub. This is separate from DCO sign-off:

| | DCO (`git commit -s`) | SSH signing (`commit.gpgsign`) |
| -- | -- | -- |
| Purpose | Legal contribution statement | Cryptographic proof of authorship |
| Required | Yes — CI enforces via DCO check | Recommended — GitHub Verified badge |

### One-time repo setup

Register your SSH public key on GitHub as a **Signing key**, then configure this repo
(repo-local settings in `.git/config`, not committed):

```bash
bash scripts/setup-git-signing.sh
```

Or manually:

```bash
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519.pub
git config commit.gpgsign true
```

If `~/.ssh/id_ed25519.pub` is missing but the private key exists:

```bash
ssh-keygen -y -f ~/.ssh/id_ed25519 > ~/.ssh/id_ed25519.pub
```

Optional local verification:

```bash
git config gpg.ssh.allowedSignersFile ~/.config/git/allowed_signers
```

When `commit.gpgsign=true`, signing is automatic — still use `-s` for DCO on every
commit. Do not add hooks that silently append sign-off; the repo enforces DCO via the
`commit-msg` hook (`scripts/git-hooks/check-dco.sh`).

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/) with a scope and, when
applicable, a linked GitHub issue:

```text
<type>(<scope>)[#<issue>]: <imperative summary>
```

| Part | Rule |
| ---- | ---- |
| `type` | `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`, `build`, or `perf` |
| `scope` | One of: `api`, `web`, `mobile`, `shared`, `infra`, `docs`, `roadmap`, `ci` |
| `[#<issue>]` | GitHub issue number for task-linked work |
| Summary | Imperative mood; whole subject ≤ 72 characters |

Examples:

```text
feat(api)[#42]: add schedule rotation resolver
fix(web)[#51]: correct timezone label on on-call calendar
chore(infra)[#33]: add AGPL license and DCO workflow
```

## Allure test reports (local)

Escalite uses [Allure Report 3](https://allurereport.org/docs/v3/) for unified HTML test
reporting. Configuration lives in `allurerc.mjs` at the repo root; generated artifacts are
gitignored (`allure-results/`, `allure-report/`).

After running tests with Allure instrumentation (see child tasks in the reporting epic), use
this workflow locally:

```bash
pnpm test:allure    # run tests and merge per-package allure-results/
pnpm report:allure  # merge results and generate the HTML report
pnpm open:allure    # serve the report in your browser
```

Equivalent [Task](https://taskfile.dev/) targets: `task test:allure`, `task report:allure`,
`task open:allure`.

Smoke-check the CLI after installing dependencies:

```bash
npx allure --version
pnpm report:allure   # succeeds even when allure-results/ is empty
```

### Node.js unit tests (`node:test`)

JS workspace packages run unit tests through `scripts/node-test-allure.mjs`, which enables the
Allure reporter (`--test-reporter allure-node-test/reporter`) and writes per-package results to
`allure-results/js/<package>/` (for example `apps/web/allure-results/js/web/`).

| Node.js version | Mode | Behavior |
| --------------- | ---- | -------- |
| **22** (CI default) | Reporter-only | Pass, fail, skip, and todo results; no `--import allure-node-test/setup` |
| **26.1+** | Full Runtime API | Also preloads `allure-node-test/setup` for `allure-js-commons` steps, labels, and attachments |

`packages/config` (noop) and `packages/schema` (lint-only) are intentionally excluded.

### Go unit tests (`go test`)

Go modules run through `scripts/go-test.sh`, which exports `ALLURE_RESULTS_DIR` (default:
`allure-results/go`) before invoking `gotestsum`.

Migrate `_test.go` files incrementally to [allure-go](https://github.com/allure-framework/allure-go):

1. Add module dependencies:

   ```bash
   go get github.com/allure-framework/allure-go/commons/gotest \
     github.com/allure-framework/allure-go/testify
   ```

2. Replace testify imports with the Allure proxy packages (same API):

   ```diff
   - "github.com/stretchr/testify/assert"
   - "github.com/stretchr/testify/require"
   + "github.com/allure-framework/allure-go/testify/assert"
   + "github.com/allure-framework/allure-go/testify/require"
   ```

3. Wrap each test function with `allure.Wrap` (one Go test → one Allure result) or
   `allure.Test` (one Go test → multiple named Allure results):

   ```go
   import allure "github.com/allure-framework/allure-go/commons/gotest"

   func TestExample(t *testing.T) {
       allure.Wrap(t, func(a *allure.Context) {
           assert.Equal(a, expected, actual)
           require.NoError(a, err)
       })
   }
   ```

   Pass the Allure context `a` to assertion calls so each assertion is reported as an Allure
   step. Helpers that need `*testing.T` can call `a.T()`. Passing `t` or `a.T()` to assertions
   keeps plain testify behavior without step reporting.

### CI reports and GitHub Pages

Every CI run (PR and branch pushes) uploads an **`allure-report`** workflow artifact you can
download from the GitHub Actions run summary. Report generation uses `if: always()` so results
are collected even when test jobs fail.

| Workflow | Allure artifact | gh-pages publish |
| -------- | --------------- | ---------------- |
| **PR** (`pr.yml`) | Yes — download from the run | No — artifact only |
| **`dev` push** (`dev.yml`) | Yes | Yes — updates canonical history |
| **`main` push** (`main.yml`) | Yes | No |

On pushes to `dev`, CI merges `allure-results` from the JavaScript, Go, and E2E jobs, loads
trend history from the `gh-pages` branch, generates the report, and publishes it via
[`peaceiris/actions-gh-pages`](https://github.com/peaceiris/actions-gh-pages). GitHub Pages
source is configured automatically (`gh-pages` branch, `/` root).

**Published report:** [https://mdg-labs.github.io/escalite/](https://mdg-labs.github.io/escalite/)

In CI, `allurerc.mjs` reads history from the checked-out `gh-pages` tree via the
`ALLURE_HISTORY_PATH` environment variable (default locally: `./.allure/history.jsonl`).

## Pull requests

1. Branch from `dev` (or the integration branch named in the issue).
2. Keep changes focused; link the related GitHub issue in the PR description.
3. Ensure every commit is signed off (`git commit -s`).
4. Open the PR against `dev` and wait for required checks (including DCO) to pass.

## License

By contributing, you agree that your contributions are licensed under the
[GNU Affero General Public License v3.0](LICENSE).
