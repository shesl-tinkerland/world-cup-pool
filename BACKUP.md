# Backup and Restore

World Cup Pool stores application state in PocketBase under `/pb_data` inside the container. That data can include users, emails, leagues, chat messages, predictions, avatars, OAuth configuration, and admin settings.

Never commit backups or extracted PocketBase data to git.

## Backup a Running Docker Instance

The default compose setup names the production container `fhun_tips` for backwards compatibility.

```sh
mkdir -p backups

docker run --rm --volumes-from fhun_tips -v "$PWD/backups":/backup alpine \
  tar czf /backup/pb_data-backup-$(date +%Y%m%d-%H%M%S).tgz -C /pb_data .
```

On PowerShell:

```powershell
New-Item -ItemType Directory -Force backups | Out-Null
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
docker run --rm --volumes-from fhun_tips -v "${PWD}/backups:/backup" alpine tar czf "/backup/pb_data-backup-$stamp.tgz" -C /pb_data .
```

## Verify a Backup

List the archive contents before relying on it:

```sh
tar tzf backups/pb_data-backup-YYYYMMDD-HHMMSS.tgz | head
```

At minimum you should see PocketBase data files and collection storage.

## Restore Into an Empty Instance

Stop the app first:

```sh
docker compose down
```

Create or clear the target volume, then extract the backup into it. A safe pattern is to restore into a new temporary volume first, inspect it, and only then attach it to the app.

Example using the default compose project volume name from your current folder:

```sh
PROJECT=$(basename "$PWD")
VOLUME="${PROJECT}_pb_data"

docker volume create "$VOLUME"
docker run --rm -v "$VOLUME":/pb_data -v "$PWD/backups":/backup alpine \
  tar xzf /backup/pb_data-backup-YYYYMMDD-HHMMSS.tgz -C /pb_data

docker compose up -d
```

If you use a custom compose project name, replace the volume name accordingly.

## Test Restores

Use a separate test container and port when validating backups:

```powershell
.\scripts\restore-test.ps1 -BackupPath .\backups\pb_data-backup-YYYYMMDD-HHMMSS.tgz -Port 8092
```

Do not restore a backup directly into production until you have tested it in an isolated container.

## Public Repo Safety

The following files must stay out of git:

- `.env`
- `pb_data/` and `pb_data_*` folders
- `*.db`, `*.sqlite`, `*.sqlite3`
- `*.tgz`, `*.tar.gz`, `*.zip` backup archives
- `*.dpapi.json` encrypted local secret exports
- Logs and local smoke-test data

If a backup or secret was ever committed to a public repository, rotate the affected credentials and create a fresh clean repository if necessary.
