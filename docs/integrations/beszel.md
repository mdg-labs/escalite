# Beszel integration

Beszel is a lightweight self-hosted server monitor. It sends alerts through [Shoutrrr](https://shoutrrr.nickfedor.com/) using the `generic://` webhook service with `template=json`.

Escalite handles Beszel via the **generic-webhook** inbound plugin — there is no dedicated Beszel backend plugin. The web UI provides a **Beszel preset** that pre-fills the field mapping (`title` → alert title, `message` → alert body).

## Prerequisites

- An Escalite service with an admin account
- Beszel installed and configured with alert thresholds

## 1. Create a Beszel integration key in Escalite

1. Sign in to Escalite as an org admin.
2. Open **Integrations** (`/integrations`).
3. Enter the target **Service ID** (UUID of the service that should receive Beszel alerts).
4. Select **Beszel** and click **Create integration key**.
5. Copy the **Beszel notification URL** shown after creation. The full webhook token is only displayed once.

Escalite creates an `IntegrationKey` with:

- `plugin_name`: `generic-webhook`
- `config`: `{"title":"title","body":"message","dedup_key":"title"}`

## 2. Configure Beszel notifications

In Beszel, open **Settings → Notifications** and add a webhook notification.

Paste the **exact** Shoutrrr URL format:

```
generic://https://your-escalite.example.com/webhook/generic-webhook/<token>?template=json
```

Replace:

- `https://your-escalite.example.com` with your Escalite API public URL (the host that serves `/webhook/...`).
- `<token>` with the integration key token from step 1.

### Example

If your Escalite API is at `https://alerts.example.com` and your token is `abc123xyz`, use:

```
generic://https://alerts.example.com/webhook/generic-webhook/abc123xyz?template=json
```

## Payload shape

With `template=json`, Shoutrrr sends a JSON body like:

```json
{
  "title": "Foo CPU above threshold",
  "message": "CPU usage is 95% on host foo"
}
```

Escalite maps these fields to alert title and description. Because Beszel does not send a separate deduplication ID, the preset uses the `title` field as `dedup_key` — repeated alerts with the same title collapse on the same service.

## Local development

When running Escalite locally, Beszel must reach the API directly (not the Vite dev server). Use your API base URL, typically:

```
generic://http://localhost:8080/webhook/generic-webhook/<token>?template=json
```

Set `VITE_API_PUBLIC_URL=http://localhost:8080` in the web app environment so the integration UI generates the correct URL.

## Related

- [Generic webhook plugin](../../services/integrations/genericwebhook/plugin.go)
- Spec: integration presets in doc 02 (Outline)
