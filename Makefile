.PHONY: run test test-integration cover lint up down logs

TEST_DATABASE_URL ?= postgres://hospital:hospital@localhost:5433/hospital_test?sslmode=disable

run: ## Run the API locally (needs DATABASE_URL and JWT_SECRET)
	go run ./cmd/server

test: ## Unit tests (no database needed)
	go test -race -cover ./...

test-integration: ## Unit + repository tests against a throwaway Postgres on :5433
	docker run -d --rm --name hm-test-db -e POSTGRES_USER=hospital -e POSTGRES_PASSWORD=hospital \
		-e POSTGRES_DB=hospital_test -p 5433:5432 postgres:16-alpine
	@until docker exec hm-test-db pg_isready -U hospital >/dev/null 2>&1; do sleep 1; done
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -cover ./... ; status=$$?; \
		docker stop hm-test-db >/dev/null; exit $$status

cover: ## HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	gofmt -l . && go vet ./...

up: ## Build and start nginx + app + postgres + mock HIS
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f app
