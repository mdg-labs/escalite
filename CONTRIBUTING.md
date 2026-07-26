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
commit. Do not add auto-sign-off hooks.

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

## Pull requests

1. Branch from `dev` (or the integration branch named in the issue).
2. Keep changes focused; link the related GitHub issue in the PR description.
3. Ensure every commit is signed off (`git commit -s`).
4. Open the PR against `dev` and wait for required checks (including DCO) to pass.

## License

By contributing, you agree that your contributions are licensed under the
[GNU Affero General Public License v3.0](LICENSE).
