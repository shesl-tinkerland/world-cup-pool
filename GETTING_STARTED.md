# Getting Started

This guide gets a fresh World Cup Pool instance running locally with Docker.

## 1. Clone

```sh
git clone https://github.com/oyvhov/world-cup-pool.git
cd world-cup-pool
```

## 2. Configure

```sh
cp .env.example .env
```

Edit `.env` and replace the placeholders:

```env
HTTP_PORT=8090
WMP_DEV=0
RESULTS_SOURCE=auto
API_FOOTBALL_KEY=
PB_ADMIN_EMAIL=admin@example.com
PB_ADMIN_PASSWORD=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

Notes:

- `API_FOOTBALL_KEY` is optional. Leave it empty for openfootball/manual results.
- Google OAuth is optional. Leave both Google fields empty for email/password auth only.
- Use a strong unique `PB_ADMIN_PASSWORD` for any real instance.

## 3. Start

```sh
docker compose up --build -d
```

Open the app at `http://localhost:8090`.

PocketBase admin is available at `http://localhost:8090/_/`.

## 4. Create Your First League

1. Create or log in to an account in the app.
2. Open `Leagues`.
3. Create a private league.
4. Share the invite code or `/join/<code>` link with friends.

Every user also joins the shared Global league automatically.

## 5. Admin Tasks

Create a superuser if needed:

```sh
docker compose exec app wm-pickems superuser create you@example.com 'a-strong-password' --dir=/pb_data
```

Admin tasks include:

- Manual result override.
- Result sync refresh.
- Scoring recompute after scoring configuration changes.
- Mail/SMTP configuration for password resets.

See [DEPLOY.md](DEPLOY.md) for details.

## 6. Local Test Harness

For development and demos, run the isolated test app on port 8091:

```powershell
.\scripts\start-test.ps1 -Port 8091
```

This does not touch the default production container or data volume.

When `WMP_DEV=1`, the `/dev` page can simulate tournament time, generate bot players, and send league chat messages from existing test bots.
