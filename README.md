# How's Working There — API

REST API backend for the How's Working There iOS application, built with [GoFr](https://gofr.dev) and PostgreSQL via Supabase.

## Tech Stack

- **Framework:** GoFr (Go microservice framework)
- **Database:** PostgreSQL via Supabase (native dialect support)
- **Auth:** Auth0 JWT validation (RS256)
- **Containerization:** Docker
- **Deployment:** Digital Ocean App Platform via GitHub Actions

## Project Structure

```
├── main.go                    # App bootstrap + route registration
├── configs/
│   ├── .env                   # Local config (gitignored)
│   └── .env.example           # Template
├── models/                    # Request/response structs
├── handler/                   # Route handlers
│   ├── location.go            # GET /api/v1/locations/search
│   ├── checkin.go             # POST check-in endpoints
│   └── views.go               # GET view-based endpoints
├── middleware/
│   └── auth.go                # Auth0 JWT middleware
├── errors/
│   └── errors.go              # Custom HTTP error types
├── Dockerfile
├── docker-compose.yml
├── .do/app.yaml               # Digital Ocean App Platform spec
└── .github/workflows/
    └── deploy.yml             # CI/CD pipeline
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/locations/search` | Find location by name, address, or geo-coordinates |
| `POST` | `/api/v1/check-ins` | Full check-in: upsert location + tests + ratings |
| `POST` | `/api/v1/locations/{location_id}/check-ins` | Check-in at existing location |
| `GET` | `/api/v1/users/{user_id}/locations` | User's tested locations (averaged) |
| `GET` | `/api/v1/locations` | All tested locations (averaged) |

All endpoints require a valid Auth0 JWT Bearer token.

## Local Development

### Prerequisites

- Go 1.22+
- Docker (optional, for containerized runs)
- A Supabase project with the schema applied (see `docs/check-in-schema.sql`)
- An Auth0 tenant configured for the iOS app

### Setup

1. Copy the environment template and fill in your credentials:
   ```bash
   cp configs/.env.example configs/.env
   ```

2. Update `configs/.env` with your Supabase and Auth0 values:
   - `SUPABASE_PROJECT_REF` — your Supabase project reference
   - `DB_PASSWORD` — your Supabase database password
   - `AUTH0_DOMAIN` — your Auth0 tenant domain
   - `AUTH0_AUDIENCE` — your Auth0 API audience

3. Apply the database schema to your Supabase project:
   ```bash
   psql "postgresql://postgres:YOUR_PASSWORD@db.YOUR_REF.supabase.co:5432/postgres" -f docs/check-in-schema.sql
   ```

4. Run the API:
   ```bash
   go run main.go
   ```

   Or with Docker:
   ```bash
   docker compose up --build
   ```

5. Verify `pgvector` is enabled (Docker local DB):
   ```bash
   docker compose exec db psql -U hwt -d howsworkingthere -c "SELECT extname FROM pg_extension WHERE extname = 'vector';"
   ```

The API starts on port `8080` by default.

## Deployment

### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `DIGITALOCEAN_ACCESS_TOKEN` | DO personal access token |
| `DIGITALOCEAN_REGISTRY_NAME` | DO container registry name |
| `DIGITALOCEAN_APP_ID` | DO App Platform app ID |

### Environment Variables (set in DO App Platform)

| Variable | Description |
|----------|-------------|
| `SUPABASE_PROJECT_REF` | Supabase project reference |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |
| `AUTH0_DOMAIN` | Auth0 tenant domain |
| `AUTH0_AUDIENCE` | Auth0 API audience |

### Deploy

Push to `main` to trigger the GitHub Actions workflow, which:
1. Runs tests
2. Builds and pushes Docker image to DO Container Registry
3. Triggers a new deployment on DO App Platform
