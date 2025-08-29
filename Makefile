.PHONY: run dev tidy build docker-up docker-down

tidy:
	go mod tidy

build:
	go build -o bin/api ./cmd/api

run:
	APP_ENV=dev HTTP_PORT=8080 DATABASE_URL="postgres://postgres:postgres@localhost:5432/insider?sslmode=disable" \
	go run ./cmd/api

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v