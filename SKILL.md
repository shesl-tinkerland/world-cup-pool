---
name: World Cup Pool
description: |
  Deploy and run a self-hosted World Cup 2026 prediction pool for friends,
  family, or your community. Private leagues, live standings, match tips,
  full-tournament bracket predictions, and league chat — all from one
  Docker image.
---

# World Cup Pool

> Provenance: shaped from [oyvhov/world-cup-pool](https://github.com/oyvhov/world-cup-pool)
> (Svelte + Go/PocketBase, GPL-3.0, 26 stars). Based on
> [floholz/wm-pickems](https://github.com/floholz/wm-pickems).
> The original repo is a fully-featured self-hosted web app but ships no
> SKILL.md. This skill wraps the deployment, configuration, and
> administration workflow into a guided conversational interface.

## What This Skill Does

World Cup Pool helps you stand up and manage a private World Cup 2026
prediction game. The app ships as a single Docker image (Go/PocketBase
backend + embedded SvelteKit frontend on one port). This skill guides you
through:

- **Deployment** — clone, configure `.env`, and launch with
  `docker compose up --build -d`. Covers Docker, VPS, and reverse-proxy
  setups.
- **Configuration** — set admin credentials, choose a result source
  (openfootball auto-sync, API-Football, or manual), enable optional
  Google OAuth, configure SMTP for password resets and signup alerts,
  set up Web Push notifications and VAPID keys, and configure the
  optional Odds API for bookmaker win-probability display.
- **League management** — create private leagues with invite codes and
  shareable `/join/<code>` links; explain the global league, leaderboard
  breakdowns, tiebreaker rules, league progress trend views, and league
  chat.
- **Match tips and predictions** — walk through the full-tournament
  prediction flow: group picks, best-third advancement, knockout bracket,
  and tournament winner; explain the editable-until-kickoff rule.
- **Scoring and admin** — explain the built-in scoring system, admin
  result override, scoring recompute after rule changes, and the
  admin panel at `/_/`.
- **Backup and restore** — use the built-in `BACKUP.md` workflow for
  PocketBase data, and the isolated test harness for staging.
- **Localization** — Bokmaal/Nynorsk/English language toggle and
  light/dark theming.

## Required Inputs

The user must provide:

- A Docker-capable host (local machine, VPS, or cloud instance)
- Admin email and password for PocketBase superuser

Optional inputs:

- `API_FOOTBALL_KEY` — for live result sync from API-Football
- `ODDS_API_KEY` — for bookmaker win-probability display (free tier: 500 req/month)
- Google OAuth credentials — for Google sign-in
- SMTP configuration — for password resets and email notifications
- VAPID keys — for Web Push notifications
- `PUBLIC_APP_URL` — public origin for social sharing and notification links

## Output Contract

The skill returns:

- Step-by-step deployment instructions tailored to the user's environment
- Correctly populated `.env` configuration (never with real secrets in
  output; always instructs the user to fill placeholders)
- Docker Compose commands for build, start, stop, and test
- Admin CLI commands (`wm-pickems superuser create`, result sync, scoring
  recompute)
- League setup guidance with invite code generation
- Troubleshooting steps for common issues (port conflicts, SMTP, OAuth)

## How To Use

Ask in natural language. Examples:

- "Help me deploy World Cup Pool on my VPS"
- "How do I configure Google OAuth for the pool?"
- "Create a private league and share the invite link"
- "Set up email notifications for new signups"
- "How does the scoring system work?"
- "Walk me through the full-tournament prediction flow"
- "Help me set up a reverse proxy with nginx"
- "How do I back up and restore the database?"

## Quick Start

```sh
git clone https://github.com/oyvhov/world-cup-pool.git
cd world-cup-pool
cp .env.example .env
# Edit .env with your admin credentials
docker compose up --build -d
# Open http://localhost:8090
```

## Source References

- Repository: https://github.com/oyvhov/world-cup-pool
- Getting Started: https://github.com/oyvhov/world-cup-pool/blob/main/GETTING_STARTED.md
- Deployment Guide: https://github.com/oyvhov/world-cup-pool/blob/main/DEPLOY.md
- Backup Guide: https://github.com/oyvhov/world-cup-pool/blob/main/BACKUP.md
- Contributing: https://github.com/oyvhov/world-cup-pool/blob/main/CONTRIBUTING.md
- Frontend (SvelteKit): https://github.com/oyvhov/world-cup-pool/tree/main/frontend
- Backend (Go): https://github.com/oyvhov/world-cup-pool/blob/main/main.go
- Docker Compose: https://github.com/oyvhov/world-cup-pool/blob/main/docker-compose.yml
