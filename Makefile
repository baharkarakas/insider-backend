.PHONY: tidy build run up down restart logs ps app-logs db-logs psql seed schema idx test-balance test-credit test-debit test-transfer token

# ---- Go ----
tidy:
	go mod tidy

build:
	go build -o bin/api ./cmd/api

run:
	APP_ENV=dev HTTP_PORT=8080 DATABASE_URL="postgres://postgres:postgres@localhost:5432/insider?sslmode=disable" \
	go run ./cmd/api

# ---- Docker ----
up:
	docker compose up -d --build

down:
	docker compose down -v

restart:
	docker compose down
	docker compose up -d --build

logs:
	docker compose logs -f

app-logs:
	docker compose logs -f app

db-logs:
	docker compose logs -f db

ps:
	docker compose ps

# ---- DB helpers ----
psql:
	docker compose exec -T db psql -U postgres -d insider

schema:
	docker compose exec -T db psql -U postgres -d insider -c "\dt+"

idx:
	docker compose exec -T db psql -U postgres -d insider -c "SELECT indexname,indexdef FROM pg_indexes WHERE schemaname='public' AND tablename='transactions';"

seed:
	# demo user (idempotent)
	docker compose exec -T db psql -U postgres -d insider -v ON_ERROR_STOP=1 -c \
	"INSERT INTO users (username,email,password_hash,role)
	 SELECT 'demo','demo@example.com','x','user'
	 WHERE NOT EXISTS (SELECT 1 FROM users WHERE email='demo@example.com');"

# ---- Quick test calls (Git Bash / WSL) ----
token:
	@USER_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo@example.com';"); \
	echo "dev-$$USER_ID"

test-balance:
	@USER_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo@example.com';"); \
	curl -s -H "Authorization: Bearer dev-$$USER_ID" http://localhost:8080/api/v1/balances/current

test-credit:
	@USER_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo@example.com';"); \
	curl -s -X POST http://localhost:8080/api/v1/transactions/credit \
	 -H "Content-Type: application/json" \
	 -H "Authorization: Bearer dev-$$USER_ID" \
	 -H "Idempotency-Key: demo-1" \
	 -d '{"amount":500}'

test-debit:
	@USER_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo@example.com';"); \
	curl -s -X POST http://localhost:8080/api/v1/transactions/debit \
	 -H "Content-Type: application/json" \
	 -H "Authorization: Bearer dev-$$USER_ID" \
	 -H "Idempotency-Key: demo-2" \
	 -d '{"amount":200}'

test-transfer:
	@A_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo@example.com';"); \
	B_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "SELECT id FROM users WHERE email='demo2@example.com';"); \
	if [ -z "$$B_ID" ]; then \
	  B_ID=$$(docker compose exec -T db psql -U postgres -d insider -t -A -c "INSERT INTO users (username,email,password_hash,role) VALUES ('demo2','demo2@example.com','x','user') RETURNING id;"); \
	fi; \
	curl -s -X POST http://localhost:8080/api/v1/transactions/transfer \
	 -H "Content-Type: application/json" \
	 -H "Authorization: Bearer dev-$$A_ID" \
	 -H "Idempotency-Key: xfer-1" \
	 -d "$$(printf '{"'"to_user_id"':"'"%s"'","'"amount"'":100}' "$$B_ID")"
