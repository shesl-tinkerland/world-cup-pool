# World Cup Pool

World Cup Pool is a self-hosted World Cup 2026 prediction game for friends,
families, teams, and communities. Players predict the full tournament, submit
match tips, and compete in private leagues with live standings.

The app ships as one Docker image: a Go/PocketBase backend serving the API and
an embedded SvelteKit frontend from the same port.

## Screenshots

![Frontpage](frontend/static/screenshots/Frontpage.png?v=20260524)

![Home Quarterfinals](frontend/static/github/Home_Quarterfinals.png?v=20260524)

## Attribution

This project is based on [floholz/wm-pickems](https://github.com/floholz/wm-pickems)
and remains licensed under GPL-3.0. The public repo is maintained as
**World Cup Pool** for easier self-hosting and community use.

Some internal binary, container, and module names still use `wm-pickems` for
backwards compatibility with existing deployments.

## Features

- Match tips for every World Cup 2026 match, editable until kickoff.
- Full-tournament predictions: groups, best thirds, knockout bracket, and winner.
- Private leagues with invite codes and shareable join links.
- Global league, leaderboard breakdowns, and tiebreaker details.
- Built-in search for matches, teams, groups, and leagues.
- Bokmål/Nynorsk/English language toggle plus light/dark theming.
- League progress and points trend views to follow score changes over time.
- League chat, friend tip visibility after kickoff, and forecast detail views.
- Admin result override and scoring recompute endpoints.
- Optional Google OAuth, email/password auth, password reset, avatars, and PWA install.
- Results can come from openfootball data, API-Football, or manual admin updates.

![Matches](frontend/static/screenshots/Matches.png?v=20260524)

## Quick Start With Docker

```sh
git clone https://github.com/oyvhov/world-cup-pool.git
cd world-cup-pool
cp .env.example .env
```

Edit `.env` before running:

- Set `PB_ADMIN_EMAIL` and `PB_ADMIN_PASSWORD` to your own admin credentials.
- Leave `API_FOOTBALL_KEY` empty unless you have a usable API-Football key.
- Leave Google OAuth fields empty unless you want Google sign-in.
- Keep `WMP_DEV=0` for normal deployments.

Start the app:

```sh
docker compose up --build -d
```

Open:

- App: `http://localhost:8090`
- PocketBase admin UI: `http://localhost:8090/_/`

Create a superuser if you did not bootstrap one from `.env`:

```sh
docker compose exec app wm-pickems superuser create you@example.com 'a-strong-password' --dir=/pb_data
```

## Local Development

```sh
make install
make dev-backend
make dev-frontend
```

The frontend dev server proxies API calls to the local backend.

Run checks:

```sh
make test
cd frontend && npm run check
```

Run the isolated Docker test app:

```powershell
.\scripts\start-test.ps1 -Port 8091
```

This uses a separate test container and volume from production.

## Configuration

Use `.env.example` as the template for local configuration. Never commit a real
`.env` file, API key, OAuth secret, password, backup archive, or PocketBase data.

Important environment variables:

| Variable | Required | Notes |
|---|---:|---|
| `HTTP_PORT` | no | Host port for Docker, defaults to `8090`. |
| `WMP_DEV` | no | Set `1` only for local simulation tools. |
| `RESULTS_SOURCE` | no | `auto`, `openfootball`, or `apifootball`. |
| `API_FOOTBALL_KEY` | no | Optional external result API key. |
| `ODDS_API_KEY` | no | Optional The Odds API key for bookmaker odds on upcoming matches. |
| `PB_ADMIN_EMAIL` | no | Optional initial PocketBase admin email. |
| `PB_ADMIN_PASSWORD` | no | Optional initial PocketBase admin password. |
| `PUBLIC_STATS_TOKEN` | no | Optional bearer token that enables `GET /api/public/stats` for Home Assistant or other monitoring. |
| `GOOGLE_CLIENT_ID` | no | Optional Google OAuth client ID. |
| `GOOGLE_CLIENT_SECRET` | no | Optional Google OAuth secret. |

## Scoring

Match tips can score up to 6 points:

| Rule | Points |
|---|---:|
| Correct result or knockout advancer | 3 |
| Exact score | +1 |
| Correct total goals | +1 |
| Correct goal difference | +1 |

Tournament predictions score group placements, perfect groups, advancing teams,
knockout reach, finalists, and champion picks. The scoring weights live in the
PocketBase `scoring_configs` records and can be changed without a redeploy.

## Operations

- Full deployment guide: [DEPLOY.md](DEPLOY.md)
- Backup and restore notes: [BACKUP.md](BACKUP.md)
- Local production/test workflow notes: [Prod_guide.md](Prod_guide.md)
- Public onboarding: [GETTING_STARTED.md](GETTING_STARTED.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)

## Data Sources

- Fixtures and initial tournament structure are seeded from openfootball data.
- Live results can be synced from API-Football when configured.
- Upcoming match odds sync from The Odds API when `ODDS_API_KEY` is configured; otherwise the app shows FIFA-ranking-based probabilities.
- Results can always be entered or corrected manually by an admin.

## Security Notes

Before publishing or deploying your own instance:

- Keep `.env` private.
- Use your own strong admin password.
- Rotate any credentials that were ever accidentally shared.
- Do not commit `pb_data`, backup archives, SQLite databases, DPAPI blobs, logs,
  or production snapshots.
- Run a secret scanner such as Gitleaks before pushing a public release.

## License

GPL-3.0. See [LICENSE](LICENSE).
