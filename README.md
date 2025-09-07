# Insider Backend – Quickstart

## Quick Start

```bash
cp .env.example .env        # JWT_* ve DB URL’lerini ayarla (opsiyonel)
docker compose down -v
docker compose up -d --build
./scripts/demo.sh           # smoke test
```

## Postman Collection

Proje kökünde `insider-backend.postman_collection.json` dosyası vardır.  
Bunu Postman içine import ederek API uçlarını hızlıca test edebilirsiniz.

Varsayılan `{{access_token}}` değişkenini `Login` endpointinden aldığınız token ile doldurmayı unutmayın.
