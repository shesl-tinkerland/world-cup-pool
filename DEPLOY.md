# Deploy

World Cup Pool ships as **one self-contained Docker image**: the Go binary
serves the API and the embedded SvelteKit SPA from a single port, with SQLite
data on a mounted volume.

For backwards compatibility, the Docker image still exposes the `wm-pickems`
binary name and the default compose file still uses the `fhun_tips` container
name. Do not change those names in an existing production instance unless you
also plan a controlled migration.

## 1. Configure

```sh
cp .env.example .env
```

| Var | Needed | Notes |
|-----|--------|-------|
| `HTTP_PORT` | no | Host port (default `8090`). |
| `PUBLIC_APP_URL` | recommended | Public origin used for invite Open Graph images and URLs, for example `https://vm.midttunet.no`. If omitted, the app derives it from forwarded headers / host. |
| `API_FOOTBALL_KEY` | optional | Leave empty unless you have your own API-Football key. |
| `RESULTS_SOURCE` | no | `auto` (default): API-Football if its key reaches WC2026, else the free **openfootball** JSON. Force with `apifootball` / `openfootball`. Manual override always works. openfootball is community-updated (hours, not real-time). |
| `PB_ADMIN_EMAIL` / `PB_ADMIN_PASSWORD` | optional | Convenience only. Use your own values, never the example values. See superuser step below. |
| `PUBLIC_STATS_TOKEN` | optional | Enables `GET /api/public/stats` for external monitoring. Request it with `Authorization: Bearer <token>`. Leave empty to disable the endpoint. |

## 2. Run

```sh
docker compose up --build -d
```

App + API: `http://<host>:${HTTP_PORT}`. Data persists in the `pb_data`
Docker volume (SQLite DB, uploaded files, logs). First boot auto-runs
migrations and seeds 48 teams / 12 groups / 104 fixtures.

## 3. Create an admin (superuser)

The PocketBase admin UI (`/_/`) and the admin endpoints
(`/api/sync/refresh`, `/api/admin/matches/{id}/result`,
`/api/admin/recompute`) require a superuser:

```sh
docker compose exec app wm-pickems superuser create you@example.com 'a-strong-pass' --dir=/pb_data
```

## 4. Configure mail / password reset

Password-reset requests use PocketBase's built-in auth email flow. The app
route and API can work without SMTP, but a real reset email is only delivered
after mail is configured:

1. Open `http://<host>:${HTTP_PORT}/_/` and sign in as the superuser.
2. Go to **Settings → Application** and set the public Application URL.
3. Go to **Settings → Mail**, set sender name/address, enable SMTP, and enter
  your SMTP host, port, username/password, TLS and auth method.
4. Use **Send test email** in PocketBase. Treat this as the delivery proof.
5. Then test the app flow from `/forgot-password`: the API should return `204`,
  the inbox should receive a link to `/confirm-password-reset/<token>`, and
  that page should accept a new password.

Without SMTP, a `/forgot-password` request can still return `204` because
PocketBase intentionally does not reveal whether an address exists, but no
external email delivery has been proven.

### Signup email alerts

When SMTP is configured, the backend sends a notification email to the admin
whenever a new user account is created (both email/password and Google OAuth).

Set the recipient in `.env`:

```env
SIGNUP_ALERT_EMAIL=you@example.com
```

If `SIGNUP_ALERT_EMAIL` is not set, it falls back to `PB_ADMIN_EMAIL`. If
neither is set, no alert is sent. Failed delivery is logged but never blocks
the signup. Dev-bot accounts (suffix `@dev.local`) are skipped.

### User notifications (opt-in)

Users choose which notifications they receive under **Settings → Notifications**,
and a one-time popup on first app open offers to turn them on. Everything is
**opt-in** (off by default), stored per user in `users.notifyPrefs`. Two
channels are available — **email** and **Web Push** — and two notification
types:

- `pre_kickoff_reminder` — sent once, ~1 day before the first match, only to
  users who have not submitted everything (all group tips **and** a complete
  forecast: group order, ≥8 best-thirds, full knockout bracket and a golden
  boot pick).
- `upcoming_matches_not_tipped` — recurring reminder when matches kick off
  within ~24h that the user hasn't tipped (deduped to at most once per day).

#### Automatic sending (cron)

Sending is driven by a 15-minute cron, off by default. Enable it once verified:

```env
NOTIFY_CRON_ENABLED=1
```

The send log (`notification_sends`) has a unique `(dedupKey, channel)` index, so
a restart or repeated cron tick never double-sends. The cron honours the dev
clock, so a test container with a simulated date behaves correctly.

#### Web Push (VAPID)

Push needs a VAPID keypair. If `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` are
unset, the app generates one on first boot and stores it in `app_meta` (so it
survives restart and is carried by a backup/restore). **In production, set them
explicitly** so the keys are stable regardless of the database:

```env
VAPID_PUBLIC_KEY=...
VAPID_PRIVATE_KEY=...
# PUSH_SUBJECT=mailto:admin@example.com   # defaults to the SMTP sender address
```

The browser uses the app's `/service-worker.js` for both offline/PWA behavior
and push events, and stores each subscription in `push_subscriptions`. Dead
subscriptions are pruned automatically when the push service returns 404/410.

#### Admin verification

Admin-only endpoints (require a superuser auth token) help verify delivery
without waiting for the schedule or touching real users:

- `POST /api/notifications/preview` — render an email (`{ "event": "pre_kickoff_reminder", "lang": "nb" }`) without sending.
- `POST /api/notifications/test` — send a test to an address (`{ "event": "...", "to": "you@example.com", "lang": "nb" }`); `to` defaults to the caller's email.
- `POST /api/notifications/send-incomplete` — send the pre-kickoff email immediately to every unfinished user by email, ignoring opt-in prefs for that explicit admin action while reusing the normal dedup key.
- `POST /api/notifications/run` — run one dispatch pass immediately (respects prefs, the pre-kickoff window and the send log).

For one-off production sends from the Windows host, use the helper scripts:

```powershell
.\scripts\send-unfinished-reminder.ps1
.\scripts\register-unfinished-reminder-task.ps1 -RunAtNorway '2026-06-11 16:00'
```

The scripts read `PB_ADMIN_EMAIL` and `PB_ADMIN_PASSWORD` from `.env`, log in as
the PocketBase superuser, and call the admin endpoint locally on
`http://localhost:8090` unless you override `-BaseUrl`.

For a Linux host, use the shell helper:

```sh
./scripts/send-unfinished-reminder.sh
```

To schedule a one-off send for `16:00` Norway/Germany time from cron on a host
that may not already run in that timezone, set an explicit cron timezone:

```cron
CRON_TZ=Europe/Berlin
0 16 11 6 * cd /srv/world-cup-pool && ./scripts/send-unfinished-reminder.sh >> /var/log/fotballvm-unfinished-reminder.log 2>&1
```

If that host deploys from GitHub or GHCR releases, deploy the new version first
so `/api/notifications/send-incomplete` exists before cron fires.

Email links use `PUBLIC_APP_URL` (falling back to the PocketBase Application URL).

## 5. Operating

- **Results**: synced every 30 min from the active source (openfootball by
  default, or a paid API-Football). Force one: `POST /api/sync/refresh`
  (superuser) — returns the source used.
- **Odds**: when `ODDS_API_KEY` is set, bookmaker odds sync at startup and
  then daily at `07:00 UTC` and `18:30 UTC` (`20:30` CEST during the
  tournament). Without a key, the tips UI falls back to FIFA-ranking-based
  probabilities.
- **Manual override / fix a result**: `POST /api/admin/matches/{id}/result`
  with `{ "FTHome":2, "FTAway":1, "Status":"finished" }` (also `ETHome/ETAway`,
  `PenHome/PenAway` for knockout). Scores recompute automatically.
- **Recompute everything** (after changing a scoring config):
  `POST /api/admin/recompute` (superuser).
- **Scoring config**: edit the `scoring_configs` "Default" record in `/_/`
  (or a per-League override) — no redeploy. Note: a config change (or a
  schema migration that rewrites it) does **not** retro-rescore matches that
  are already finished until you call `POST /api/admin/recompute` (or the
  next result comes in, which recomputes automatically).

## 6. Backup

The whole app state is the `/pb_data` volume. Snapshot it while running:

```sh
docker run --rm --volumes-from fhun_tips -v "$PWD":/backup alpine \
  tar czf /backup/pb_data-backup.tgz -C /pb_data .
```

Restore by extracting the archive back into an empty data volume before `up`.
Keep backups outside git. They can contain users, emails, league data, chat,
avatars, and secrets.

## 7. TLS / reverse proxy

Terminate TLS at a proxy (Caddy/Traefik/nginx) and forward to the container
port. Example Caddy:

```
pickems.example.com {
    reverse_proxy localhost:8090
}
```

## 8. Updating

```sh
git pull
docker compose up --build -d   # migrations run automatically on boot
```

## Health

`GET /api/health` returns 200 when up — use it for container/proxy health
checks.

## Home Assistant

If you want Home Assistant to track how many users each deployment has, set
`PUBLIC_STATS_TOKEN` in each app's `.env` and restart the container. Then Home
Assistant can poll the token-protected JSON endpoint:

```yaml
rest:
  - resource: https://vmpool.app/api/public/stats
    headers:
      Authorization: Bearer your-shared-token
    scan_interval: 300
    sensor:
      - name: VMPool Users
        value_template: "{{ value_json.users.total }}"
        unit_of_measurement: users
      - name: VMPool Verified Users
        value_template: "{{ value_json.users.verified }}"
        unit_of_measurement: users

  - resource: https://vm.midttunet.no/api/public/stats
    headers:
      Authorization: Bearer your-shared-token
    scan_interval: 300
    sensor:
      - name: Midttunet Users
        value_template: "{{ value_json.users.total }}"
        unit_of_measurement: users
```

Optional combined sensor:

```yaml
template:
  - sensor:
      - name: World Cup Pool Users Total
        unit_of_measurement: users
        state: >-
          {{ states('sensor.vmpool_users') | int(0)
             + states('sensor.midttunet_users') | int(0) }}
```

The endpoint excludes local dev-bot accounts (`@dev.local`) from the totals.
