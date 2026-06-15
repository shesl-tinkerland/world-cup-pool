# Production and Test Workflow

This repository keeps production and test workflows separate. The default production port is `8090`; the helper test scripts refuse to use that port.

For backwards compatibility with existing deployments, the default production container is named `fhun_tips` and the Docker image is tagged `wm-pickems:latest`.

## Production Safety Rules

- Do not run test helpers on port `8090`.
- Do not point test containers at the production `/pb_data` volume.
- Do not rename the production container or image in an existing deployment unless you plan a controlled migration.
- Always take a backup before deploying a new image.
- Verify a backup restore in an isolated test container before relying on it.
- **Never set `WMP_DEV=1` in production.** When enabled it disables all prediction-lock enforcement, allows arbitrary clock manipulation via the database, and exposes dev-only endpoints. If you suspect it is set, check with `docker inspect fhun_tips | grep WMP_DEV` and remove it immediately.

## Build Production Image

```powershell
docker compose build
```

## Start or Update Production

```powershell
docker compose up -d
```

Check status:

```powershell
docker ps --filter "name=^/fhun_tips$"
docker logs --tail 100 fhun_tips
```

Open the app at `http://localhost:8090` or through your reverse proxy.

## Isolated Test App

Start the test app on a non-production port:

```powershell
.\scripts\start-test.ps1 -Port 8091
```

Stop it:

```powershell
.\scripts\stop-test.ps1
```

The test app uses the `fhun_tips_test` container and a separate `fhun_tips_test_pb_data_test` volume.

## Test a Backup Restore

```powershell
.\scripts\restore-test.ps1 -BackupPath .\backups\pb_data-backup-YYYYMMDD-HHMMSS.tgz -Port 8092
```

This starts an isolated restore container and does not touch production.

## Notable API Behaviour

| Endpoint | Notes |
|----------|-------|
| `GET /api/leagues/{id}/leaderboard` | Now includes `rankDelta` per row: positive = moved up since last matchday, negative = dropped. The delta is computed from the most recently finalized calendar day's match scores. |
| `GET /api/tips/others/{matchId}` | Now includes the requesting user's own tip first (`isMe: true`) and a `points` field (−1 = no tip submitted) on every row. Only available after kickoff. |
| `GET /api/sync/refresh` (superuser) | The football API client now returns an explicit error if the server reports `results > 0` but the decoded response array is empty, catching silent schema drift before scores are zeroed. |

## Useful Checks

Frontend checks:

```powershell
Set-Location frontend
npm run check
```

Go tests through Docker when Go is not installed locally:

```powershell
docker run --rm -v "${PWD}:/app" -w /app golang:1.26-alpine sh -c "go test ./..."
```

Verify results-source team-name coverage (API-Football path only):

The default openfootball sync maps matches by a deterministic `ExtID` and needs
no team-name aliasing. The optional API-Football path (paid plan) matches by
team name, so before relying on it confirm every team maps. With a dev build
(`WMP_DEV=1`, isolated container) call the diagnostic:

```
GET /api/dev/apicheck?season=2026   # or ?season=2022 to validate the parse path
```

Inspect `unmappedTeams` and `ourMatchesMapped`; add any missing spellings to
`nameAliases` in `internal/sync/sync.go`.

Secret scan before publishing:

```powershell
gitleaks detect --no-git
```

If Gitleaks is not installed, run a targeted text scan and inspect every hit before publishing.

## Sync Public Repo

```powershell
.\scripts\sync-public.ps1
```

- Requires a clean working tree.
- Exports the current private `HEAD` as a fresh snapshot repo and force-pushes public `main`, so the public repo keeps one current-state commit rather than the private commit history.
- By default the public commit message matches the current private `HEAD` commit message. Use `-CommitMessage` to override it.
