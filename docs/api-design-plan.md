# API Design Plan — How's Working There (GoFr + PostgreSQL + Docker)

> **Framework:** [GoFr](https://gofr.dev) (Go opinionated microservice framework)
> **Database:** PostgreSQL via Supabase (GoFr native `supabase` dialect)
> **Auth:** Auth0 JWT (Bearer token, `sub` claim = `user_id`)
> **Containerization:** Docker
> **Deployment:** Digital Ocean App Platform via GitHub Actions CI/CD

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Endpoints](#2-endpoints)
3. [Data Models (Go Structs)](#3-data-models-go-structs)
4. [Handler Specifications](#4-handler-specifications)
5. [Database Access Layer](#5-database-access-layer)
6. [Auth Middleware](#6-auth-middleware)
7. [Docker Setup](#7-docker-setup)
8. [Configuration](#8-configuration)
9. [Error Handling](#9-error-handling)
10. [Implementation Order](#10-implementation-order)

---

## 1. Project Structure

```
hows-working-there-api/
├── main.go                         # App bootstrap, route registration
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── configs/
│   └── .env                        # GoFr env config (DB, ports, etc.)
├── docs/                           # Existing specs
│   ├── check-in-api-model.md
│   ├── check-in-schema.sql
│   ├── add-location-spec.md
│   ├── authentication-spec.md
│   └── api-design-plan.md          # This file
├── models/
│   ├── location.go                 # Location struct + request model
│   ├── checkin.go                  # CheckIn request/response structs
│   ├── speedtest.go                # SpeedTest + ServerInfo structs (NOT speed_test.go — Go treats _test.go as test files)
│   ├── noise_level.go              # NoiseLevel struct
│   ├── workspace_rating.go         # WorkspaceRating struct
│   └── views.go                    # Structs for view responses (v_location_averages, v_user_location_averages)
├── handler/
│   ├── location.go                 # Location search/find handlers
│   ├── checkin.go                  # Full check-in creation handler
│   └── views.go                    # View query handlers (user locations, all locations)
├── middleware/
│   └── auth.go                     # Auth0 JWT validation middleware (JWKS-based RS256)
├── errors/
│   └── errors.go                   # Custom HTTP error types (GoFr StatusCode() pattern)
├── .do/
│   └── app.yaml                    # Digital Ocean App Platform spec
└── .github/workflows/
    └── deploy.yml                  # CI/CD: test → build → push to DOCR → deploy
```

---

## 2. Endpoints

All endpoints are prefixed with `/api/v1`.

### 2.1 Location Search

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/api/v1/locations/search` | Find existing location by name, address, or geo-coordinates | Yes |

**Query Parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Partial match on location name (case-insensitive) |
| `address` | string | No | Partial match on address (case-insensitive) |
| `latitude` | float64 | No | Center latitude for geo search |
| `longitude` | float64 | No | Center longitude for geo search |
| `radius_km` | float64 | No | Search radius in km (default: 0.5) |

At least one of `name`, `address`, or (`latitude` + `longitude`) must be provided.

**Example Requests:**

```
GET /api/v1/locations/search?name=Surin
GET /api/v1/locations/search?address=810+N+Highland
GET /api/v1/locations/search?latitude=33.7901&longitude=-84.3513&radius_km=1.0
```

**Response: `200 OK`**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Surin Of Thailand",
      "address": "810 N Highland Ave NE Atlanta GA 30306",
      "latitude": 33.7901,
      "longitude": -84.3513,
      "category": "restaurant_bar",
      "mapkit_poi_category": "MKPOICategoryRestaurant"
    }
  ]
}
```

---

### 2.2 Create Full Check-In (New Location)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/api/v1/check-ins` | Insert a new location with speed test, noise level, and workspace ratings | Yes |

This is the primary endpoint called from the iOS "Add Location" flow. It atomically:
1. Inserts the location (or finds existing via `uq_locations_name_coords` unique constraint)
2. Creates the check-in record
3. Creates the speed test record
4. Creates the noise level record
5. Creates the workspace rating record

All within a single database transaction.

**Request Body:** Matches the payload in `check-in-api-model.md`:

```json
{
  "id": "uuid",
  "user_id": "auth0|abc123",
  "timestamp": "2026-03-03T20:49:00Z",
  "location": {
    "name": "Surin Of Thailand",
    "address": "810 N Highland Ave NE Atlanta GA 30306",
    "latitude": 33.7901,
    "longitude": -84.3513,
    "category": "restaurant_bar",
    "mapkit_poi_category": "MKPOICategoryRestaurant"
  },
  "speed_test": { ... },
  "noise_level": { ... },
  "workspace_ratings": { ... }
}
```

**Response: `201 Created`**

```json
{
  "data": {
    "check_in_id": "uuid",
    "location_id": "uuid",
    "location_is_new": true,
    "created_at": "2026-03-03T20:49:00Z"
  }
}
```

**Idempotency:** The client-generated `id` field is used as the check-in primary key. If a duplicate `id` is submitted, the API returns the existing record rather than creating a duplicate (HTTP `200` instead of `201`).

---

### 2.3 Create Check-In at Existing Location

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/api/v1/locations/{location_id}/check-ins` | Insert a new check-in (speed test, noise level, ratings) for an existing location | Yes |

This endpoint is for when the iOS client has already identified an existing location (via the search endpoint) and wants to add a new round of tests.

**Request Body:**

```json
{
  "id": "uuid",
  "user_id": "auth0|abc123",
  "timestamp": "2026-03-03T21:15:00Z",
  "speed_test": {
    "download_speed_mbps": 147.16,
    "upload_speed_mbps": 22.91,
    "latency_ms": 14,
    "jitter": 3.75,
    "isp_name": "AT&T Wireless",
    "ip_address": "10.0.0.1",
    "network_type": "wifi",
    "packet_loss_percent": 0.0,
    "time_to_first_byte_ms": 42,
    "download_transferred_mb": 85.3,
    "upload_transferred_mb": 22.1,
    "server": {
      "domain": "speedtest-atl.example.com",
      "city": "Atlanta",
      "country": "United States"
    },
    "speed_test_id": "64C4FB29-3BEF-4C3B-AF59-C65D5A287FB3",
    "skipped": false
  },
  "noise_level": {
    "average_decibels": 52.3,
    "peak_decibels": 68.7,
    "duration_seconds": 10.0,
    "skipped": false
  },
  "workspace_ratings": {
    "outlets_at_bar": true,
    "outlets_at_table": false,
    "crowdedness": 2,
    "ease_of_work": 1,
    "best_work_type": "solo"
  }
}
```

**Response: `201 Created`**

```json
{
  "data": {
    "check_in_id": "uuid",
    "location_id": "uuid",
    "created_at": "2026-03-03T21:15:00Z"
  }
}
```

---

### 2.4 User's Tested Locations (View)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/api/v1/users/{user_id}/locations` | Return all locations tested by a specific user with averaged scores | Yes |

Queries `v_user_location_averages` filtered by `user_id`.

**Security:** The `user_id` path parameter must match the authenticated user's `sub` claim (users can only view their own data via this endpoint).

**Response: `200 OK`**

```json
{
  "data": [
    {
      "location_id": "uuid",
      "location_name": "Surin Of Thailand",
      "location_address": "810 N Highland Ave NE Atlanta GA 30306",
      "latitude": 33.7901,
      "longitude": -84.3513,
      "location_category": "restaurant_bar",
      "my_check_ins": 3,
      "my_avg_download_mbps": 142.50,
      "my_avg_upload_mbps": 21.30,
      "my_avg_latency_ms": 15,
      "my_avg_jitter": 3.50,
      "my_speed_test_count": 3,
      "my_avg_decibels": 51.0,
      "my_avg_peak_decibels": 67.5,
      "my_noise_test_count": 3,
      "my_avg_crowdedness": 2.0,
      "my_avg_ease_of_work": 1.3,
      "my_rating_count": 3,
      "my_pct_outlets_at_bar": 67,
      "my_pct_outlets_at_table": 33,
      "my_most_common_work_type": "solo",
      "my_first_check_in": "2026-01-15T10:00:00Z",
      "my_last_check_in": "2026-03-03T20:49:00Z"
    }
  ]
}
```

---

### 2.5 All Tested Locations (View)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/api/v1/locations` | Return all tested locations with averaged scores across all users | Yes |

Queries `v_location_averages`.

**Query Parameters (optional):**

| Param | Type | Description |
|-------|------|-------------|
| `category` | string | Filter by location category |
| `latitude` | float64 | Center latitude for geo bounding |
| `longitude` | float64 | Center longitude for geo bounding |
| `radius_km` | float64 | Bounding radius in km |
| `limit` | int | Max results (default: 50) |
| `offset` | int | Pagination offset (default: 0) |

**Response: `200 OK`**

```json
{
  "data": [
    {
      "location_id": "uuid",
      "location_name": "Surin Of Thailand",
      "location_address": "810 N Highland Ave NE Atlanta GA 30306",
      "latitude": 33.7901,
      "longitude": -84.3513,
      "location_category": "restaurant_bar",
      "total_check_ins": 12,
      "unique_users": 5,
      "avg_download_mbps": 135.20,
      "avg_upload_mbps": 20.10,
      "avg_latency_ms": 16,
      "avg_jitter": 4.00,
      "speed_test_count": 10,
      "avg_decibels": 53.0,
      "avg_peak_decibels": 70.2,
      "noise_test_count": 11,
      "avg_crowdedness": 2.1,
      "avg_ease_of_work": 1.5,
      "rating_count": 12,
      "pct_outlets_at_bar": 75,
      "pct_outlets_at_table": 42,
      "most_common_work_type": "solo",
      "first_check_in": "2025-11-01T08:00:00Z",
      "last_check_in": "2026-03-03T20:49:00Z"
    }
  ]
}
```

---

## 3. Data Models (Go Structs)

### 3.1 Request Models

```go
// CheckInRequest matches the iOS client payload from check-in-api-model.md
type CheckInRequest struct {
    ID        string    `json:"id" validate:"required,uuid"`
    UserID    string    `json:"user_id" validate:"required"`
    Timestamp time.Time `json:"timestamp" validate:"required"`

    Location        LocationRequest        `json:"location" validate:"required"`
    SpeedTest       SpeedTestRequest       `json:"speed_test" validate:"required"`
    NoiseLevel      NoiseLevelRequest      `json:"noise_level" validate:"required"`
    WorkspaceRatings WorkspaceRatingsRequest `json:"workspace_ratings" validate:"required"`
}

type LocationRequest struct {
    Name              string  `json:"name" validate:"required"`
    Address           string  `json:"address" validate:"required"`
    Latitude          float64 `json:"latitude" validate:"required,min=-90,max=90"`
    Longitude         float64 `json:"longitude" validate:"required,min=-180,max=180"`
    Category          string  `json:"category" validate:"required,oneof=cafe restaurant_bar hotel park library office coworking other"`
    MapkitPOICategory *string `json:"mapkit_poi_category"`
}

type SpeedTestRequest struct {
    DownloadSpeedMbps    float64      `json:"download_speed_mbps"`
    UploadSpeedMbps      float64      `json:"upload_speed_mbps"`
    LatencyMs            int          `json:"latency_ms"`
    Jitter               float64      `json:"jitter"`
    ISPName              *string      `json:"isp_name"`
    IPAddress            *string      `json:"ip_address"`
    NetworkType          string       `json:"network_type" validate:"required,oneof=wifi cellular unknown"`
    PacketLossPercent    *float64     `json:"packet_loss_percent"`
    TimeToFirstByteMs    int          `json:"time_to_first_byte_ms"`
    DownloadTransferredMB float64     `json:"download_transferred_mb"`
    UploadTransferredMB  float64      `json:"upload_transferred_mb"`
    Server               *ServerInfo  `json:"server"`
    SpeedTestID          string       `json:"speed_test_id" validate:"required"`
    Skipped              bool         `json:"skipped"`
}

type ServerInfo struct {
    Domain  *string `json:"domain"`
    City    *string `json:"city"`
    Country *string `json:"country"`
}

type NoiseLevelRequest struct {
    AverageDecibels float64 `json:"average_decibels"`
    PeakDecibels    float64 `json:"peak_decibels"`
    DurationSeconds float64 `json:"duration_seconds"`
    Skipped         bool    `json:"skipped"`
}

type WorkspaceRatingsRequest struct {
    OutletsAtBar  bool   `json:"outlets_at_bar"`
    OutletsAtTable bool  `json:"outlets_at_table"`
    Crowdedness   int    `json:"crowdedness" validate:"required,min=1,max=3"`
    EaseOfWork    int    `json:"ease_of_work" validate:"required,min=1,max=3"`
    BestWorkType  string `json:"best_work_type" validate:"required,oneof=solo team both"`
}

// ExistingLocationCheckInRequest — for POST /locations/{location_id}/check-ins
type ExistingLocationCheckInRequest struct {
    ID               string                  `json:"id" validate:"required,uuid"`
    UserID           string                  `json:"user_id" validate:"required"`
    Timestamp        time.Time               `json:"timestamp" validate:"required"`
    SpeedTest        SpeedTestRequest        `json:"speed_test" validate:"required"`
    NoiseLevel       NoiseLevelRequest       `json:"noise_level" validate:"required"`
    WorkspaceRatings WorkspaceRatingsRequest `json:"workspace_ratings" validate:"required"`
}
```

### 3.2 DB / Response Models

```go
type Location struct {
    ID                string    `json:"id" db:"id"`
    Name              string    `json:"name" db:"name"`
    Address           string    `json:"address" db:"address"`
    Latitude          float64   `json:"latitude" db:"latitude"`
    Longitude         float64   `json:"longitude" db:"longitude"`
    Category          string    `json:"category" db:"category"`
    MapkitPOICategory *string   `json:"mapkit_poi_category,omitempty" db:"mapkit_poi_category"`
    CreatedAt         time.Time `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type UserLocationAverage struct {
    // Maps directly to v_user_location_averages view columns
    UserID                string   `json:"user_id" db:"user_id"`
    LocationID            string   `json:"location_id" db:"location_id"`
    LocationName          string   `json:"location_name" db:"location_name"`
    LocationAddress       string   `json:"location_address" db:"location_address"`
    Latitude              float64  `json:"latitude" db:"latitude"`
    Longitude             float64  `json:"longitude" db:"longitude"`
    LocationCategory      string   `json:"location_category" db:"location_category"`
    MyCheckIns            int      `json:"my_check_ins" db:"my_check_ins"`
    MyAvgDownloadMbps     *float64 `json:"my_avg_download_mbps" db:"my_avg_download_mbps"`
    MyAvgUploadMbps       *float64 `json:"my_avg_upload_mbps" db:"my_avg_upload_mbps"`
    MyAvgLatencyMs        *float64 `json:"my_avg_latency_ms" db:"my_avg_latency_ms"`
    MyAvgJitter           *float64 `json:"my_avg_jitter" db:"my_avg_jitter"`
    MySpeedTestCount      int      `json:"my_speed_test_count" db:"my_speed_test_count"`
    MyAvgDecibels         *float64 `json:"my_avg_decibels" db:"my_avg_decibels"`
    MyAvgPeakDecibels     *float64 `json:"my_avg_peak_decibels" db:"my_avg_peak_decibels"`
    MyNoiseTestCount      int      `json:"my_noise_test_count" db:"my_noise_test_count"`
    MyAvgCrowdedness      *float64 `json:"my_avg_crowdedness" db:"my_avg_crowdedness"`
    MyAvgEaseOfWork       *float64 `json:"my_avg_ease_of_work" db:"my_avg_ease_of_work"`
    MyRatingCount         int      `json:"my_rating_count" db:"my_rating_count"`
    MyPctOutletsAtBar     *float64 `json:"my_pct_outlets_at_bar" db:"my_pct_outlets_at_bar"`
    MyPctOutletsAtTable   *float64 `json:"my_pct_outlets_at_table" db:"my_pct_outlets_at_table"`
    MyMostCommonWorkType  *string  `json:"my_most_common_work_type" db:"my_most_common_work_type"`
    MyFirstCheckIn        *string  `json:"my_first_check_in" db:"my_first_check_in"`
    MyLastCheckIn         *string  `json:"my_last_check_in" db:"my_last_check_in"`
}

type LocationAverage struct {
    // Maps directly to v_location_averages view columns
    LocationID           string   `json:"location_id" db:"location_id"`
    LocationName         string   `json:"location_name" db:"location_name"`
    LocationAddress      string   `json:"location_address" db:"location_address"`
    Latitude             float64  `json:"latitude" db:"latitude"`
    Longitude            float64  `json:"longitude" db:"longitude"`
    LocationCategory     string   `json:"location_category" db:"location_category"`
    TotalCheckIns        int      `json:"total_check_ins" db:"total_check_ins"`
    UniqueUsers          int      `json:"unique_users" db:"unique_users"`
    AvgDownloadMbps      *float64 `json:"avg_download_mbps" db:"avg_download_mbps"`
    AvgUploadMbps        *float64 `json:"avg_upload_mbps" db:"avg_upload_mbps"`
    AvgLatencyMs         *float64 `json:"avg_latency_ms" db:"avg_latency_ms"`
    AvgJitter            *float64 `json:"avg_jitter" db:"avg_jitter"`
    SpeedTestCount       int      `json:"speed_test_count" db:"speed_test_count"`
    AvgDecibels          *float64 `json:"avg_decibels" db:"avg_decibels"`
    AvgPeakDecibels      *float64 `json:"avg_peak_decibels" db:"avg_peak_decibels"`
    NoiseTestCount       int      `json:"noise_test_count" db:"noise_test_count"`
    AvgCrowdedness       *float64 `json:"avg_crowdedness" db:"avg_crowdedness"`
    AvgEaseOfWork        *float64 `json:"avg_ease_of_work" db:"avg_ease_of_work"`
    RatingCount          int      `json:"rating_count" db:"rating_count"`
    PctOutletsAtBar      *float64 `json:"pct_outlets_at_bar" db:"pct_outlets_at_bar"`
    PctOutletsAtTable    *float64 `json:"pct_outlets_at_table" db:"pct_outlets_at_table"`
    MostCommonWorkType   *string  `json:"most_common_work_type" db:"most_common_work_type"`
    FirstCheckIn         *string  `json:"first_check_in" db:"first_check_in"`
    LastCheckIn          *string  `json:"last_check_in" db:"last_check_in"`
}
```

---

## 4. Handler Specifications

### 4.1 `handler/location.go` — `SearchLocations`

```
GET /api/v1/locations/search
```

**Logic:**
1. Parse query params: `name`, `address`, `latitude`, `longitude`, `radius_km`
2. Validate that at least one search criterion is provided
3. Build dynamic SQL query against `locations` table:
   - `name`: `WHERE LOWER(name) LIKE LOWER('%' || $1 || '%')`
   - `address`: `WHERE LOWER(address) LIKE LOWER('%' || $1 || '%')`
   - Geo: Haversine distance filter `WHERE (haversine(lat, lng, $1, $2)) <= $3`
   - Combine with `AND` if multiple criteria provided
4. Return matched locations

**Haversine approximation (PostgreSQL):**
```sql
(6371 * acos(
    cos(radians($1)) * cos(radians(latitude)) *
    cos(radians(longitude) - radians($2)) +
    sin(radians($1)) * sin(radians(latitude))
)) <= $3
```

### 4.2 `handler/checkin.go` — `CreateCheckIn`

```
POST /api/v1/check-ins
```

**Logic:**
1. Bind and validate `CheckInRequest`
2. Verify `req.UserID` matches the JWT `sub` claim from the auth middleware
3. Begin a database transaction
4. **Upsert location**: `INSERT INTO locations ... ON CONFLICT (name, latitude, longitude) DO UPDATE SET updated_at = NOW() RETURNING id`
   - This handles both new and existing locations in one query
   - Returns `location_id` (and whether it was newly inserted)
5. Insert into `check_ins` with the client-supplied `id` and the resolved `location_id`
6. Insert into `speed_tests` with `check_in_id` and `location_id`
7. Insert into `noise_levels` with `check_in_id` and `location_id`
8. Insert into `workspace_ratings` with `check_in_id` and `location_id`
9. Commit transaction
10. Return `201` with check-in and location IDs

**Idempotency:** If the `check_ins.id` already exists (duplicate PK), return the existing record with `200`.

### 4.3 `handler/checkin.go` — `CreateCheckInAtLocation`

```
POST /api/v1/locations/{location_id}/check-ins
```

**Logic:**
1. Extract `location_id` from path parameter
2. Bind and validate `ExistingLocationCheckInRequest`
3. Verify `req.UserID` matches JWT `sub`
4. Verify the location exists: `SELECT id FROM locations WHERE id = $1`
5. Begin transaction
6. Insert into `check_ins`
7. Insert into `speed_tests`
8. Insert into `noise_levels`
9. Insert into `workspace_ratings`
10. Commit transaction
11. Return `201`

### 4.4 `handler/views.go` — `GetUserLocations`

```
GET /api/v1/users/{user_id}/locations
```

**Logic:**
1. Extract `user_id` from path parameter
2. Verify `user_id` matches JWT `sub` (users can only query their own data)
3. Query `v_user_location_averages` view: `SELECT * FROM v_user_location_averages WHERE user_id = $1`
4. Return results

### 4.5 `handler/views.go` — `GetAllLocations`

```
GET /api/v1/locations
```

**Logic:**
1. Parse optional query params: `category`, `latitude`, `longitude`, `radius_km`, `limit`, `offset`
2. Query `v_location_averages` view with optional filters
3. Apply geo-bounding if lat/lng provided
4. Apply pagination (`LIMIT`/`OFFSET`)
5. Return results

---

## 5. Database Access Layer

GoFr provides `c.SQL` for database operations. Key patterns:

### 5.1 Queries (SELECT)

```go
// Multiple rows
var locations []Location
err := c.SQL.Select(c, &locations, "SELECT * FROM locations WHERE LOWER(name) LIKE $1", pattern)

// Single row
var loc Location
err := c.SQL.QueryRowContext(c, "SELECT ... WHERE id = $1", id).Scan(&loc.ID, &loc.Name, ...)
```

### 5.2 Inserts (within transactions)

```go
tx, err := c.SQL.Begin()
if err != nil {
    return nil, err
}
defer tx.Rollback()

// Upsert location
var locationID string
err = tx.QueryRowContext(c,
    `INSERT INTO locations (name, address, latitude, longitude, category, mapkit_poi_category)
     VALUES ($1, $2, $3, $4, $5, $6)
     ON CONFLICT (name, latitude, longitude) DO UPDATE SET updated_at = NOW()
     RETURNING id`,
    req.Location.Name, req.Location.Address, req.Location.Latitude, req.Location.Longitude,
    req.Location.Category, req.Location.MapkitPOICategory,
).Scan(&locationID)

// Insert check-in, speed test, noise level, ratings...

err = tx.Commit()
```

### 5.3 PostgreSQL Placeholder Style

GoFr with PostgreSQL uses `$1, $2, ...` positional placeholders (not `?`).

---

## 6. Auth Middleware

### 6.1 JWT Validation

The middleware validates the Auth0 JWT Bearer token on every request:

```go
func AuthMiddleware() gofrHTTP.Middleware {
    return func(inner http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract "Bearer <token>" from Authorization header
            // 2. Validate JWT signature against Auth0 JWKS endpoint
            //    - JWKS URL: https://{AUTH0_DOMAIN}/.well-known/jwks.json
            //    - Cache the JWKS keys
            // 3. Validate claims: issuer, audience, expiration
            // 4. Extract "sub" claim as user_id
            // 5. Set user_id in request context for handlers to use
            // 6. Call inner.ServeHTTP(w, r)
        })
    }
}
```

### 6.2 Configuration

| Env Var | Description |
|---------|-------------|
| `AUTH0_DOMAIN` | Auth0 tenant domain (e.g., `dev-xxxx.us.auth0.com`) |
| `AUTH0_AUDIENCE` | API identifier registered in Auth0 |

### 6.3 Extracting User ID in Handlers

```go
func getUserIDFromContext(c *gofr.Context) string {
    return c.Request.Context().Value("user_id").(string)
}
```

Handlers that accept `user_id` in the body or path must validate it matches the JWT `sub`.

---

## 7. Docker & Deployment Setup

The database is hosted on **Supabase** (not containerized locally). The Docker image contains only the API server. Schema is applied directly to Supabase via `psql` or the Supabase SQL editor.

### 7.1 Dockerfile

Multi-stage build for minimal production image:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server main.go

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
COPY configs/ ./configs/
EXPOSE 8080
CMD ["./server"]
```

### 7.2 docker-compose.yml (local dev)

For local development, connects to Supabase remotely via `configs/.env`:

```yaml
version: "3.9"
services:
  api:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - configs/.env
```

### 7.3 Digital Ocean App Platform

App spec lives in `.do/app.yaml`. Secrets are configured as encrypted env vars in DO.

### 7.4 GitHub Actions CI/CD (`.github/workflows/deploy.yml`)

On push to `main`:
1. Checkout + set up Go 1.22
2. Run `go test ./...`
3. Build Docker image, tag with commit SHA + `latest`
4. Push to Digital Ocean Container Registry (DOCR)
5. Trigger `doctl apps create-deployment` on the App Platform app

**Required GitHub Secrets:**

| Secret | Description |
|--------|-------------|
| `DIGITALOCEAN_ACCESS_TOKEN` | DO personal access token |
| `DIGITALOCEAN_REGISTRY_NAME` | DO container registry name |
| `DIGITALOCEAN_APP_ID` | DO App Platform app ID |

---

## 8. Configuration

### 8.1 `configs/.env`

GoFr has native Supabase dialect support. Setting `DB_DIALECT=supabase` + `SUPABASE_PROJECT_REF` auto-constructs the host as `db.<ref>.supabase.co` and enforces SSL.

```dotenv
APP_NAME=hows-working-there-api
HTTP_PORT=8080

# PostgreSQL via Supabase (GoFr native supabase dialect)
DB_DIALECT=supabase
SUPABASE_PROJECT_REF=xxxxxxxxxxxxxxxxxxxx
DB_USER=postgres
DB_PASSWORD=your-supabase-db-password
DB_NAME=postgres
DB_PORT=5432
DB_SSL_MODE=require

# Auth0
AUTH0_DOMAIN=dev-xxxx.us.auth0.com
AUTH0_AUDIENCE=https://api.howsworkingthere.com
```

---

## 9. Error Handling

GoFr handlers return `(any, error)`. Consistent error responses:

| HTTP Status | When |
|-------------|------|
| `400 Bad Request` | Missing/invalid query params, malformed JSON, validation failure |
| `401 Unauthorized` | Missing or invalid JWT token |
| `403 Forbidden` | JWT valid but `user_id` mismatch (trying to access another user's data) |
| `404 Not Found` | Location not found (for `POST /locations/{id}/check-ins`) |
| `409 Conflict` | Idempotency collision with different data (edge case) |
| `500 Internal Server Error` | Database or unexpected errors |

---

## 10. Implementation Status

### Phase 1: Scaffold & Infrastructure ✅
- [x] Go module initialized with GoFr v1.54.5
- [x] `Dockerfile` (multi-stage) and `docker-compose.yml` created
- [x] `configs/.env.example` with Supabase + Auth0 settings
- [x] `.gitignore` excludes `configs/.env` and binaries

### Phase 2: Models ✅
- [x] Go structs in `models/` matching check-in-api-model.md and check-in-schema.sql
- [x] View response structs for `v_location_averages` and `v_user_location_averages`
- [x] Note: file named `speedtest.go` (not `speed_test.go` — Go treats `_test.go` as test files)

### Phase 3: Core Endpoints ✅
- [x] `POST /api/v1/check-ins` — full check-in with location upsert + transaction
- [x] `GET /api/v1/locations/search` — search by name, address, or Haversine geo-distance
- [x] `POST /api/v1/locations/{location_id}/check-ins` — check-in at existing location

### Phase 4: View Endpoints ✅
- [x] `GET /api/v1/users/{user_id}/locations` — user's tested locations from `v_user_location_averages`
- [x] `GET /api/v1/locations` — all tested locations from `v_location_averages` with filtering + pagination

### Phase 5: Auth & Security ✅
- [x] Auth0 JWT middleware with JWKS caching (RS256)
- [x] `user_id` validation (JWT `sub` vs request body/path `user_id`)
- [x] Custom error types for 400/401/403/404 via GoFr `StatusCode()` pattern

### Phase 6: Deployment ✅
- [x] `.do/app.yaml` — Digital Ocean App Platform spec
- [x] `.github/workflows/deploy.yml` — CI/CD pipeline (test → build → DOCR → deploy)
- [x] `README.md` with setup and deployment instructions

### Phase 7: Remaining (TODO)
- [ ] Apply `check-in-schema.sql` to Supabase project
- [ ] Configure Auth0 API audience and test with real tokens
- [ ] Write integration tests for each endpoint
- [ ] Add structured request validation
- [ ] Load test with sample data

---

## Endpoint Summary

| # | Method | Path | Description |
|---|--------|------|-------------|
| 1 | `GET` | `/api/v1/locations/search` | Find location by name, address, or geo |
| 2 | `POST` | `/api/v1/check-ins` | Full check-in: new location + tests + ratings |
| 3 | `POST` | `/api/v1/locations/{location_id}/check-ins` | Check-in at existing location |
| 4 | `GET` | `/api/v1/users/{user_id}/locations` | User's tested locations (averaged) |
| 5 | `GET` | `/api/v1/locations` | All tested locations (averaged) |
