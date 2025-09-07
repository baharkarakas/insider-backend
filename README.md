# Insider Backend – Quickstart

## Quick Start

```bash
cp .env.example .env        # JWT_* ve DB URL’lerini ayarla (opsiyonel)
docker compose down -v
docker compose up -d --build
./scripts/demo.sh           # smoke test
```
