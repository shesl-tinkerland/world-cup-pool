# Contributing

Thanks for improving World Cup Pool.

This project is based on `floholz/wm-pickems` and is licensed under GPL-3.0. By contributing, you agree that your contribution is distributed under the same license.

## Development Setup

Install frontend dependencies:

```sh
make install
```

Run backend and frontend separately:

```sh
make dev-backend
make dev-frontend
```

Run Go tests:

```sh
make test
```

Run frontend checks:

```sh
cd frontend
npm run check
npm test -- --run
```

Build the single binary:

```sh
make build
```

Run the Docker test app:

```powershell
.\scripts\start-test.ps1 -Port 8091
```

## Pull Request Checklist

Before opening a PR:

- Do not commit `.env`, API keys, OAuth secrets, passwords, PocketBase data, backups, logs, or local database files.
- Run a secret scanner if your change touches config, scripts, docs, or deployment files.
- Run relevant tests/checks for the files you changed.
- Keep docs and examples placeholder-only.
- Keep production and test runtime paths separate.

## Code Style

- Keep changes focused and small.
- Prefer existing project patterns over new abstractions.
- Keep user-facing text available in Bokmål, Nynorsk, and English where the UI supports all three.
- Do not add generated build artifacts to commits.

## Security

If you find a security issue, do not open a public issue with secrets or exploit details. Contact the maintainer privately first.
