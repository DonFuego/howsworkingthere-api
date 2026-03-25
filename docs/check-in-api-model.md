# Check-In API Data Model

This document defines the data model for the "Add Location" check-in feature. It serves as the contract between the iOS client and the API.

---

## Endpoint

```
POST /api/v1/check-ins
```

## Payload

```json
{
  "id": "uuid",
  "user_id": "string",
  "timestamp": "2026-03-03T20:49:00Z",

  "location": {
    "id": "uuid"
  },

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

---

## Field Reference

### Root

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | UUID | Yes | Client-generated unique ID for idempotency |
| `user_id` | String | Yes | Auth0 user ID |
| `timestamp` | ISO 8601 | Yes | When the check-in was performed |

### `location`

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | UUID | Yes | Location id from MapKit search |

### `speed_test`

| Field | Type | Required | Description |
|---|---|---|---|
| `download_speed_mbps` | Double | Yes | Download speed in Mbps |
| `upload_speed_mbps` | Double | Yes | Upload speed in Mbps |
| `latency_ms` | Int | Yes | Round-trip latency in ms |
| `jitter` | Double | Yes | Latency variance in ms |
| `isp_name` | String? | No | Internet service provider |
| `ip_address` | String? | No | Device public IP |
| `network_type` | Enum | Yes | `"wifi"` \| `"cellular"` \| `"unknown"` |
| `packet_loss_percent` | Double? | No | Packet loss as percentage |
| `time_to_first_byte_ms` | Int | Yes | TTFB in ms |
| `download_transferred_mb` | Double | Yes | Total data downloaded during test |
| `upload_transferred_mb` | Double | Yes | Total data uploaded during test |
| `server.domain` | String? | No | Test server hostname |
| `server.city` | String? | No | Test server city |
| `server.country` | String? | No | Test server country |
| `speed_test_id` | String | Yes | SpeedChecker test reference ID |
| `skipped` | Bool | Yes | `true` if user skipped this step |

### `noise_level`

| Field | Type | Required | Description |
|---|---|---|---|
| `average_decibels` | Double | Yes | Average dB SPL over measurement window |
| `peak_decibels` | Double | Yes | Peak dB SPL recorded |
| `duration_seconds` | Double | Yes | How long the measurement ran |
| `skipped` | Bool | Yes | `true` if user skipped this step |

### `workspace_ratings`

| Field | Type | Required | Description |
|---|---|---|---|
| `outlets_at_bar` | Bool | Yes | Power outlets at bar/counter seating |
| `outlets_at_table` | Bool | Yes | Power outlets at table seating |
| `crowdedness` | Int (1-3) | Yes | `1` = Empty, `2` = Somewhat Crowded, `3` = Crowded |
| `ease_of_work` | Int (1-3) | Yes | `1` = Easy, `2` = Moderate, `3` = Difficult |
| `best_work_type` | Enum | Yes | `"solo"` \| `"team"` |

---

## Enum Definitions

### `network_type`
- `"wifi"` — Connected via Wi-Fi
- `"cellular"` — Connected via cellular data
- `"unknown"` — Connection type could not be determined

### `crowdedness`
- `1` — Empty
- `2` — Somewhat Crowded
- `3` — Crowded

### `ease_of_work`
- `1` — Easy
- `2` — Moderate
- `3` — Difficult

### `best_work_type`
- `"solo"` — Best for individual work
- `"team"` — Best for group/team work

### `category`
- `"cafe"` — Coffee shop, bakery
- `"restaurant_bar"` — Restaurant, bar, brewery, winery
- `"hotel"` — Hotel lobby or business center
- `"park"` — Park or outdoor space
- `"library"` — Public or university library
- `"office"` — Traditional office space
- `"coworking"` — Coworking space
- `"other"` — Uncategorized or not listed above

---

## Notes

- **Idempotency**: The client-generated `id` prevents duplicate submissions if the request is retried.
- **Skipped steps**: When `speed_test.skipped` or `noise_level.skipped` is `true`, numeric fields will be `0` / `0.0` and optional fields will be `null`. The API should still accept the payload.
- **The `speed_test` and `noise_level` objects are always present** — the `skipped` flag distinguishes real data from defaults.
- **`download_transferred_mb` / `upload_transferred_mb`** are included from the SDK but not shown to the user; useful for server-side analytics.
- **IP address** should be treated as PII and handled accordingly in storage/retention policies.
- **Category** is auto-detected from `MKMapItem.pointOfInterestCategory` when available, then confirmed or overridden by the user. The `mapkit_poi_category` field preserves the raw MapKit value for analytics.
