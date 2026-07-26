# GitHub Actions workflows

## Entry workflows

Only these files define `on:` triggers:

| File | Triggers | Purpose |
| ---- | -------- | ------- |
| [`pr.yml`](pr.yml) | `pull_request` | CI + DCO |
| [`main.yml`](main.yml) | `push` → `main` | CI, core release draft, mobile stable draft |
| [`dev.yml`](dev.yml) | `push` → `dev` | CI, nightly container images, mobile beta |
| [`release.yml`](release.yml) | `release` → `published`; `workflow_dispatch` | Core `v*` publish — GHCR, digests, SBOMs |
| [`mobile-release.yml`](mobile-release.yml) | `release` → `published` | Mobile `mobile-v*` publish hooks |

## Reusable workflows (`workflow_call` only)

| File | Called from |
| ---- | ----------- |
| [`ci.yml`](ci.yml) | `pr.yml`, `main.yml`, `dev.yml` |
| [`dco.yml`](dco.yml) | `pr.yml` |
| [`container-images.yml`](container-images.yml) | `dev.yml` (nightly), `release.yml` (stable) |
| [`release-draft.yml`](release-draft.yml) | `main.yml` |
| [`mobile-beta.yml`](mobile-beta.yml) | `dev.yml` |
| [`mobile-stable.yml`](mobile-stable.yml) | `main.yml` |

## Release tag conventions

| Stream | Version file | Git tag |
| ------ | ------------ | ------- |
| Core containers | `VERSION` | `vX.Y.Z` |
| Mobile | `apps/mobile/package.json` | `mobile-vX.Y.Z` or `mobile-vX.Y.Z-beta.N` |

`release.yml` ignores `mobile-v*` tags. `mobile-release.yml` runs only for `mobile-v*` tags.

See [`docs/deploy/mobile-releases.md`](../docs/deploy/mobile-releases.md) for operator steps.
