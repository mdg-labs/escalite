# iOS Critical Alerts entitlement — operator runbook

**Roadmap:** `p2-ios-critical-alerts-entitlement` ([#109](https://github.com/mdg-labs/escalite/issues/109))  
**Epic:** `phase-2-ios-critical-alerts` (EL-17)  
**Spec:** [03 — Mobile App Spec (Minimal)](../specs/03-mobile-app-spec.md) (Outline: `why-native`, `notification-payload-contract`)

Apple grants the Critical Alerts entitlement (`com.apple.developer.usernotifications.critical-alerts`) per App ID after manual review. Escalite cannot ship true Critical Alerts until that approval is on the App ID. This runbook covers the **human gate** in Apple Developer portal plus the **day-one fallback** that ships without it.

## App identifiers

| Field | Value |
| ----- | ----- |
| **Bundle ID** | `dev.mdglabs.escalite` |
| **Expo slug** | `escalite` |
| **Entitlement key** | `com.apple.developer.usernotifications.critical-alerts` |
| **Config source** | `apps/mobile/app.json` → `expo.ios.entitlements` |

## In-repo configuration (automated)

The mobile app is pre-configured so that, once Apple approves the entitlement, EAS/iOS builds pick it up without further code changes:

- `apps/mobile/app.json` declares the Critical Alerts entitlement under `expo.ios.entitlements`.
- `expo-notifications` config plugin enables `enableBackgroundRemoteNotifications` for APNs background delivery.
- Push payload contract includes optional `critical: true` (`apps/mobile/src/push/payload.ts`); runtime delivery as a Critical Alert is implemented in [#110](https://github.com/mdg-labs/escalite/issues/110) after approval.

> **Build note:** iOS provisioning profiles will not include the Critical Alerts capability until Apple approves the entitlement for this Bundle ID. Expect EAS Build / `eas credentials` errors for production iOS until approval; use the Time-Sensitive fallback below for day-one push.

## Operator checklist — submit entitlement request

**Status:** ☐ **Not submitted** (no portal evidence in repo as of this runbook)

Complete every step, then record completion in the **Submission record** section at the bottom.

### Prerequisites

- [ ] Active [Apple Developer Program](https://developer.apple.com/programs/) membership (Team Agent or Admin).
- [ ] Bundle ID `dev.mdglabs.escalite` registered under **Certificates, Identifiers & Profiles → Identifiers**.

### 1. Request the entitlement from Apple

1. Open Apple's Critical Alerts request form:  
   <https://developer.apple.com/contact/request/notifications-critical-alerts-entitlement/>
2. Sign in with the Escalite Apple Developer account.
3. Provide:
   - **App name:** Escalite
   - **Bundle ID:** `dev.mdglabs.escalite`
   - **Use case:** On-call / incident alerting for platform and SRE teams — time-sensitive operational alerts that must reach responders when devices are muted or in Focus/Do Not Disturb (same category as PagerDuty, Opsgenie, ilert).
   - **Why Critical Alerts:** Standard notifications can be silenced during on-call incidents; missed pages directly impact incident response and customer uptime.
4. Submit the form and save the confirmation email or case reference.

### 2. Enable Critical Alerts on the App ID (pending approval)

After Apple acknowledges the request (approval may take days or weeks):

1. Go to [Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list).
2. Select identifier **`dev.mdglabs.escalite`**.
3. Under **Capabilities**, enable **Critical Alerts**.
4. Save. Status should show **Critical Alerts** enabled (may display as pending until Apple completes review).

### 3. Refresh iOS provisioning for EAS Build

1. In the Escalite repo: `cd apps/mobile`
2. Regenerate credentials so profiles include the new capability:
   ```bash
   eas credentials --platform ios
   ```
   Choose the production (or relevant) profile and let EAS sync with Apple.
3. Trigger a new iOS build:
   ```bash
   eas build --platform ios --profile production
   ```
4. Verify the built `.ipa` / embedded provisioning profile lists `com.apple.developer.usernotifications.critical-alerts`.

### 4. Unblock follow-up engineering ([#110](https://github.com/mdg-labs/escalite/issues/110))

- [ ] Move Phasical task **Implement Critical Alerts push when entitlement approved** to Ready.
- [ ] Comment on GitHub #109 with approval date and App ID screenshot (optional).

## Time-Sensitive notification fallback (day-one behavior)

Per doc 03 (`notification-payload-contract`, open item on Apple review posture):

| Aspect | Critical Alert (post-approval) | Time-Sensitive (fallback, ships now) |
| ------ | ------------------------------ | ------------------------------------ |
| **Entitlement** | `com.apple.developer.usernotifications.critical-alerts` required | None — standard push + Time-Sensitive interruption level |
| **Bypasses DND / mute** | Yes — plays sound even when silenced or in Focus | **No** — respects hard mute and Do Not Disturb |
| **Bypasses notification summary batching** | Yes | Yes |
| **Expo / APNs** | `interruptionLevel: "critical"` in push payload ([#110](https://github.com/mdg-labs/escalite/issues/110)) | `interruptionLevel: "time-sensitive"` (or high priority without `critical`) |
| **Payload `critical: true`** | Honored when entitlement + user permission granted | **Ignored for delivery tier** — server and app still send/parse the flag; iOS delivers as Time-Sensitive until entitlement is active |
| **Android** | High-importance channel + full-screen intent ([#110](https://github.com/mdg-labs/escalite/issues/110)) | High-importance `alerts` channel (`apps/mobile/src/push/notifications.ts`) |

**Operational guarantee:** Escalite **ships on-call push with Time-Sensitive (or default high-priority) delivery regardless of Critical Alerts approval status.** Users receive alert notifications; only the DND/mute bypass tier is gated on Apple.

**Server push (Phase 1 / basic push):** Send via Expo Push API with `priority: "high"` and, on iOS, `interruptionLevel: "time-sensitive"` for alert.triggered payloads. Do **not** set `interruptionLevel: "critical"` until #110 is verified against an approved entitlement build.

**Client (current):** `obtainExpoPushToken()` requests standard notification permission (`apps/mobile/src/push/register-device.ts`). Critical Alert permission (`allowCriticalAlerts: true`) is added in #110 after entitlement approval.

## Submission record

| Field | Value |
| ----- | ----- |
| **Entitlement request submitted** | ☐ No — _operator must complete § Operator checklist_ |
| **Submitted by** | |
| **Submitted on (UTC)** | |
| **Apple case / confirmation ref** | |
| **App ID Critical Alerts capability** | ☐ Not enabled / ☐ Pending / ☐ Approved |
| **First production iOS build with entitlement** | |

When the operator completes portal submission, update this table (or add a PR amending this file) and check the boxes in § Operator checklist.

## References

- Apple: [Critical Alerts entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.developer.usernotifications.critical-alerts)
- Apple: [Request Critical Alerts entitlement](https://developer.apple.com/contact/request/notifications-critical-alerts-entitlement/)
- Escalite Outline doc 03 — `why-native`, `notification-payload-contract`
- Roadmap: `docs/roadmap/ROADMAP.md` → `p2-ios-critical-alerts-entitlement`, `p2-ios-critical-alerts-impl` (archived; see Phasical)
