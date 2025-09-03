# Insider Backend Path – Quickstart

## Prereqs

- Go 1.22+
- Docker + Docker Compose

## Dev (local Postgres optional)

```bash
go mod tidy
make run # uses local DB URL from Makefile; or use docker-up to run stack
```

## Quick Start

````bash
# Docker
make up           # build + start
make app-logs     # API logları
make db-logs      # DB logları
make down         # stop

# Seed + token (Git Bash/WSL)
make seed
make token

# Hızlı testler
make test-balance
make test-credit
make test-debit
make test-transfer
### Windows PowerShell
```powershell
# USER_ID ayarla
$env:USER_ID = (docker compose exec -T db `
  psql -U postgres -d insider -t -A `
  -c "SELECT id FROM users WHERE email='demo@example.com';").Trim()

# Okuma
curl.exe -s http://localhost:8080/api/v1/balances/current `
  -H "Authorization: Bearer dev-$env:USER_ID" |
  ConvertFrom-Json | ConvertTo-Json -Depth 5
````
