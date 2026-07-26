# Mobile releases (Android)

Android APKs are built with **EAS Build** in GitHub Actions. iOS builds are deferred until post-MVP.

**Operator setup (Expo token, Android keystore, Obtainium):** see Outline doc [09 — Mobile Android Release Setup (operator)](https://outline.mdg-labs.dev/doc/09-mobile-android-release-setup-operator-ij3kQaiwug).

## In-app server configuration

Distributed APKs do not embed an Escalite instance URL. On first launch, users enter their public Escalite origin (for example `https://escalite.example.com`). The app probes `/healthz`, stores the origin in SecureStore, and uses it for auth, GraphQL, and push registration.

## Independent version streams

Mobile and core (containers) use **separate version sources and Git tags**. Bump only what changed.

| Stream | Version source | Semver in file | Git tag | Branch |
| ------ | -------------- | -------------- | ------- | ------ |
| Core containers | [`VERSION`](../../VERSION) | `0.1.0` | `v0.1.0` | `main` |
| Mobile beta | `apps/mobile/package.json` | `0.1.0-beta.1` | `mobile-v0.1.0-beta.1` | `dev` |
| Mobile stable | `apps/mobile/package.json` | `0.1.0` | `mobile-v0.1.0` | `main` |

The `mobile-v` prefix is applied by CI when creating Git tags — do **not** put it in `package.json`.

## Version bumps

| File | Branch | Example | CI result |
| ---- | ------ | ------- | --------- |
| `apps/mobile/package.json` | `dev` | `0.1.0-beta.1` | Pre-release `mobile-v0.1.0-beta.1` + beta APK |
| `apps/mobile/package.json` | `dev` | `0.1.0-beta.11` (after `0.1.0-beta.10`) | Pre-release `mobile-v0.1.0-beta.11` + beta APK |
| `apps/mobile/package.json` | `dev` | `0.2.0-beta.1` (after `0.1.0-beta.10`) | Pre-release `mobile-v0.2.0-beta.1` + beta APK |
| `apps/mobile/package.json` | `dev` | `0.1.0` (no `-beta.`) | No beta APK (promotion-ready) |
| `apps/mobile/package.json` | `main` | `0.1.0` | Draft `mobile-v0.1.0` + stable APK |
| `VERSION` | `main` | `0.1.0` | Draft GitHub release `v0.1.0` for containers |

Beta builds are gated by **tag / pre-release existence**, not by whether `package.json` increased on the latest commit. Set `apps/mobile/package.json` to the target prerelease semver (for example `0.1.0-beta.11` or `0.2.0-beta.1`); CI builds and publishes only when `mobile-v<semver>` does not already exist as a Git tag or GitHub pre-release. Re-push the same version to retry a failed build.

A security patch can bump only `VERSION` on `main` (new `v*` release, no mobile build). Likewise, a mobile-only fix can bump only `apps/mobile/package.json`.

## Workflows

Entry workflows (the only files with triggers) are documented in [`.github/workflows/README.md`](../../.github/workflows/README.md).

| Entry workflow | Trigger | Reusable child | Result |
| -------------- | ------- | -------------- | ------ |
| `dev.yml` | `dev` push | `mobile-beta.yml` | Beta APK + `mobile-v*` pre-release |
| `main.yml` | `main` push | `mobile-stable.yml` | Stable APK + `mobile-v*` draft |
| `main.yml` | `main` push | `release-draft.yml` | Core `v*` draft (from `VERSION`) |
| `mobile-release.yml` | `mobile-v*` published | — | Post-publish hook (APK already attached) |
| `release.yml` | `v*` published | `container-images.yml` | GHCR images + SBOMs (not for `mobile-v*`) |

Tags are created only after a successful EAS build. Beta skips when the `mobile-v*` tag or GitHub pre-release for the current `package.json` version already exists; stable skips when a published (non-draft) release already exists.

APK assets are named `escalite-mobile-<semver>.apk` (for example `escalite-mobile-0.1.0-beta.1.apk`).

## Local development

Use `EXPO_PUBLIC_ESCALITE_API_URL` / `EXPO_PUBLIC_ESCALITE_WEB_URL` for `expo start`, or configure the server in-app after installing a dev build.
