# Hospital Middleware

A Go/Gin API that lets hospital staff search patient records. Records come from each hospital's HIS (Hospital Information System), and staff only ever see patients of their own hospital.

**Stack:** Go 1.24 · Gin · PostgreSQL 16 · Nginx · Docker Compose

The full planning document covers project structure, the API spec, the ER diagram and design decisions: **[docs/DEVELOPMENT_PLAN.md](docs/DEVELOPMENT_PLAN.md)**.

## Quick start

```bash
cp .env.example .env            # then set JWT_SECRET, e.g. `openssl rand -hex 32`
docker compose up -d --build
curl http://localhost:8080/health
```

Compose starts four containers:

| Service | Purpose |
|---|---|
| `nginx` | Reverse proxy, published on `localhost:${NGINX_PORT:-8080}` |
| `app` | The Go API. Runs DB migrations and seeds demo data on start-up |
| `postgres` | PostgreSQL 16 |
| `mock-his-a` | Mock of `https://hospital-a.api.co.th` for the demo |

The seed creates two hospitals. `hospital-a` has an HIS (the mock) and `hospital-b` has none.

## Try it

```bash
# 1. Create a staff member
curl -s localhost:8080/staff/create -H 'Content-Type: application/json' \
  -d '{"username":"nurse01","password":"password123","hospital":"hospital-a"}'

# 2. Log in and keep the token
TOKEN=$(curl -s localhost:8080/staff/login -H 'Content-Type: application/json' \
  -d '{"username":"nurse01","password":"password123","hospital":"hospital-a"}' | jq -r .access_token)

# 3. Search (all filters optional; results limited to hospital-a)
curl -s "localhost:8080/patient/search?first_name=som" -H "Authorization: Bearer $TOKEN" | jq

# 4. This patient exists only in Hospital A's HIS; the middleware fetches and stores it
curl -s "localhost:8080/patient/search?national_id=1103700000033" -H "Authorization: Bearer $TOKEN" | jq

# 5. A hospital-b patient is invisible to hospital-a staff -> empty result
curl -s "localhost:8080/patient/search?passport_id=AA1234567" -H "Authorization: Bearer $TOKEN" | jq
```

`POST /patient/search` with a JSON body also works and takes the same fields.

## Tests

```bash
make test               # unit tests, no database needed
make test-integration   # also runs repository tests against a throwaway Postgres container
```

Handlers, services, the HIS client, auth and middleware are tested with fakes and `httptest`, covering positive and negative cases. The repository layer has a unit-tested SQL builder plus integration tests against real Postgres. They check hospital scoping, Thai/English name search, upsert and uniqueness rules.

| Package | Coverage |
|---|---|
| config, his, middleware, models, router, service | 100% |
| handler | 96% |
| auth | 94% |
| repository (incl. integration tests) | 96% |

## Configuration

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — (required) | Postgres connection string |
| `JWT_SECRET` | — (required, ≥ 32 chars) | HMAC key for access tokens |
| `JWT_TTL` | `24h` | Token lifetime |
| `HIS_TIMEOUT` | `5s` | Timeout for HIS API calls |
| `PORT` | `8080` | App listen port |
| `GIN_MODE` | `release` | Gin mode |
