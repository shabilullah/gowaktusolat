# Go Waktu Solat API

A Go/Gofiber prayer time API server that scrapes prayer times from [JAKIM's e-Solat](https://www.e-solat.gov.my) and serves them in a v1-compatible REST API with GPS zone auto-detection.
This API is live at [api.waktusolat.online](https://api.waktusolat.online/). Its structure and availability are provided on a best-effort basis and may change without notice.

## Quick Start

```bash
# Clone and build
git clone https://github.com/shabilullah/gowaktusolat.git
cd gowaktusolat
go build ./cmd/scraper && go build ./cmd/server

# Seed zones and scrape current year prayer times (~80s for all 53 zones).
# On first startup, the server auto-seeds zones + current year data.
# These CLI commands are for manual runs or custom DB paths:
./scraper seed-zones
./scraper scrape --year=$(date +%Y)

# Start the API server (auto-seeds on first run, scraper runs in background)
./server
# Server listening on http://localhost:8080

## Development

```bash
# Install air (one-time)
go install github.com/air-verse/air@latest

# Start with hot-reload — rebuilds on file changes
# (uses .air.toml pre-configured to build cmd/server)
air

# Or run directly (no hot-reload)
go run ./cmd/server
```

### Environment variables

All settings are read from a `.env` file at the project root (optional). Copy `.env.example` to `.env` and edit:

```bash
cp .env.example .env
```

| Variable       | Default              | Description                                      |
|----------------|----------------------|--------------------------------------------------|
| `PORT`         | `8080`               | Server listen port                               |
| `DB_PATH`      | `data/waktusolat.db` | SQLite database file path                        |
| `BASE_PATH`    | `""`                 | URL prefix (reserved)                            |
| `CORS_ORIGINS` | `*`                  | `*` for all, or comma-separated domains |
| `PREFORK`      | `false`              | Spawn one process per CPU core (`true`/`1`)    |
| `API_KEY`      | `""`                 | Protects `POST /api/cache/reset` when set; send as `X-API-Key` header |
| `SEEDER_SCHED` | `""`                 | Cron schedule for auto-scheduler; omit to disable |

**CORS examples:**

```bash
# Allow all origins (default)
CORS_ORIGINS=*

# Single domain
CORS_ORIGINS=https://myapp.com

# Multiple domains
CORS_ORIGINS=https://myapp.com,https://admin.myapp.com,https://api.myapp.com
```
### Database

| Driver | Package | CGO | Notes |
|--------|---------|-----|-------|
| SQLite | `zombiezen.com/go/sqlite` | Yes | Native API, connection pooling via `sqlitex.Pool`, WAL mode |

All code uses the `zombiezen.com/go/sqlite` native API — there is no `database/sql` abstraction. A `sqlitex.Pool` (4 connections for the server, 1 for the scraper) is shared across all handlers via `pool.Take(ctx)` / `pool.Put(conn)`. In-memory databases with `?cache=shared` are used in tests.

**JSON encoding** uses `goccy/go-json` across the board — both via `fiber.Config{JSONEncoder, JSONDecoder}` for API responses, and `client.SetJSONUnmarshal` in the scraper.

**Concurrency**: The pool serializes writes across connections while allowing concurrent reads. Writes are rare (settings, scheduled scraper) so contention is negligible. Prefork is safe — each worker process gets its own pool.

### Project layout

### Running tests

```bash
go test ./internal/...
```

## Production

### Docker

```bash
# Pull and run
docker run -d \
  --name gowaktusolat \
  -p 8080:8080 \
  -v ./data:/data \
  -e CORS_ORIGINS=* \
  -e PREFORK=false \
  -e API_KEY= \
  -e SEEDER_SCHED= \
  ghcr.io/shabilullah/gowaktusolat:master

# Or use docker compose
docker compose up -d
```

The server auto-seeds zones and current-year prayer times on first run (~80s in background).
Set `SEEDER_SCHED` to a cron expression (e.g. `0 2 1 1 *` for yearly on Jan 1 at 2am) to enable auto-scraping on schedule.
Omit it or leave empty to disable recurring scrapes.

Edit `docker-compose.yml` to change settings — all environment variables are listed inline with comments.
The `.env` file is also mounted (optional) for the Go app's built-in `.env` loader.

Manual seed/scrape from the host (against the volume-mounted DB):

```bash
go run ./cmd/scraper seed-zones
go run ./cmd/scraper scrape --year=$(date +%Y)
```

Images are published automatically to `ghcr.io/shabilullah/gowaktusolat` on every push to `master` and on version tags (`v1.0.0` → `:1.0`, `:1`, `:latest`).

### Binary

```bash
# Requires CGO (zombiezen.com/go/sqlite wraps native SQLite).
# Cross-compile for Linux AMD64:
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o scraper ./cmd/scraper
```

### Systemd

1. Copy `server`, `scraper` binaries and `data/` directory to the target host.
2. Seed zones and scrape initial data (or skip — the server auto-seeds on first start):
   ```bash
   ./scraper seed-zones
   ./scraper scrape --year=$(date +%Y)
   ```
3. Create the systemd unit:
   ```ini
   # /etc/systemd/system/gowaktusolat.service
   [Unit]
   Description=Go Waktu Solat API
   After=network.target

   [Service]
   Type=simple
   User=nobody
   WorkingDirectory=/opt/gowaktusolat
   Environment=PORT=8080
   Environment=DB_PATH=/opt/gowaktusolat/data/waktusolat.db
   Environment=SEEDER_SCHED=0 2 1 1 *
   ExecStart=/opt/gowaktusolat/server
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```
4. Enable and start:
   ```bash
   sudo systemctl enable --now gowaktusolat
   ```
   Set `SEEDER_SCHED` in the unit file to enable recurring scrapes. No runtime configuration is needed — the schedule is read from the environment on startup.

### Reverse proxy (nginx)

```nginx
server {
    listen 80;
    server_name solat.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## API Reference

All endpoints return `Content-Type: application/json` unless noted. Cached routes include `Cache-Control: public, max-age=3600`.

### Zones

<details>
<summary><code>GET /api/zones</code> — List all zones</summary>

```
GET /api/zones
```

**Response** `200` — array of zone objects

```json
[
  {"jakimCode": "JHR01", "negeri": "Johor", "daerah": "Pulau Aur dan Pulau Pemanggil"},
  {"jakimCode": "SGR01", "negeri": "Selangor", "daerah": "Gombak, Petaling, Sepang, Hulu Langat, Hulu Selangor, Shah Alam"}
]
```

| Field       | Type     | Description          | Example                                        |
|-------------|----------|----------------------|------------------------------------------------|
| `jakimCode` | `string` | JAKIM zone code      | `"SGR01"`                                      |
| `negeri`    | `string` | State name (Malay)   | `"Selangor"`                                   |
| `daerah`    | `string` | District(s) covered  | `"Gombak, Petaling, Sepang, Hulu Langat, ..."` |

</details>

<details>
<summary><code>GET /api/zones/:state</code> — Filter zones by state prefix</summary>

```
GET /api/zones/SGR
```

**Response** `200` — filtered array

```json
[
  {"jakimCode": "SGR01", "negeri": "Selangor", "daerah": "Gombak, Petaling, Sepang, Hulu Langat, Hulu Selangor, Shah Alam"},
  {"jakimCode": "SGR02", "negeri": "Selangor", "daerah": "Kuala Selangor, Sabak Bernam"},
  {"jakimCode": "SGR03", "negeri": "Selangor", "daerah": "Klang, Kuala Langat"}
```

**Fields** — same as [`GET /api/zones`](#zones) above (all three-letter codes match the `:state` prefix).

</details>

<details>
<summary><code>GET /api/zones/:lat/:long</code> — Detect zone by GPS coordinate</summary>

```
GET /api/zones/3.068498/101.630263
```

**Response** `200` — detected zone

```json
{"zone": "SGR01", "state": "SGR", "district": "Petaling"}
```

**Fields**

| Field      | Type     | Description            | Example      |
|------------|----------|------------------------|--------------|
| `zone`     | `string` | Detected JAKIM zone    | `"SGR01"`    |
| `state`    | `string` | Three-letter code      | `"SGR"`      |
| `district` | `string` | Matched district name  | `"Petaling"` |

**Errors**

| Status | Body | Cause |
|--------|------|-------|
| `422` | `{"message":"Invalid latitude"}` | Non-numeric lat/long |
| `404` | `{"message":"No zone found for the given coordinates"}` | Point outside Malaysia |

</details>

### Prayer Times (Solat)

<details>
<summary><code>GET /api/solat/:zone</code> — Month of prayer times</summary>

```
GET /api/solat/SGR01?month=6&year=2026
```

**Query parameters**

| Param   | Type   | Default        | Constraints |
|---------|--------|----------------|-------------|
| `month` | `int`  | current month  | 1–12        |
| `year`  | `int`  | current year   | >= 2020     |

**Response** `200`

```json
{
  "prayerTime": [
    {
      "hijri": "1447-12-15",
      "date": "01-Jun-2026",
      "day": "Monday",
      "imsak": "05:39:00",
      "fajr": "05:49:00",
      "syuruk": "07:01:00",
      "dhuha": "07:26:00",
      "dhuhr": "13:14:00",
      "asr": "16:39:00",
      "maghrib": "19:22:00",
      "isha": "20:37:00"
    }
  ],
  "status": "OK!",
  "serverTime": "2026-06-26 20:34:11",
  "periodType": "month",
  "lang": "",
  "zone": "SGR01",
  "bearing": ""
}

**Wrapper fields**

| Field        | Type     | Description                                      | Example                     |
|--------------|----------|--------------------------------------------------|-----------------------------|
| `prayerTime` | `array`  | List of daily prayer time objects (see below)    | `[{"hijri":"…","date":"…"}]` |
| `status`     | `string` | `"OK!"` on success                               | `"OK!"`                     |
| `serverTime` | `string` | Server timestamp in `YYYY-MM-DD HH:MM:SS`        | `"2026-06-26 20:34:11"`     |
| `periodType` | `string` | `"month"` for month queries, `"day"` for day     | `"month"`                   |
| `lang`       | `string` | Reserved, always `""`                            | `""`                        |
| `zone`       | `string` | JAKIM zone code queried                          | `"SGR01"`                   |
| `bearing`    | `string` | Reserved, always `""`                            | `""`                        |

**`prayerTime[]` fields**

| Field     | Type     | Description                                           | Example           |
|-----------|----------|-------------------------------------------------------|-------------------|
| `hijri`   | `string` | Hijri date in `YYYY-MM-DD` format                     | `"1447-12-15"`    |
| `date`    | `string` | Gregorian date in `DD-Mon-YYYY` format                | `"01-Jun-2026"`   |
| `day`     | `string` | Weekday name in English (`"Monday"`–`"Sunday"`)       | `"Monday"`        |
| `imsak`   | `string` | Imsak time in `HH:MM:SS` (24h)                        | `"05:39:00"`      |
| `fajr`    | `string` | Fajr/Subuh time in `HH:MM:SS` (24h)                   | `"05:49:00"`      |
| `syuruk`  | `string` | Sunrise time in `HH:MM:SS` (24h)                      | `"07:01:00"`      |
| `dhuha`   | `string` | Dhuha time in `HH:MM:SS` (24h)                        | `"07:26:00"`      |
| `dhuhr`   | `string` | Dhuhr/Zohor time in `HH:MM:SS` (24h)                  | `"13:14:00"`      |
| `asr`     | `string` | Asr time in `HH:MM:SS` (24h)                          | `"16:39:00"`      |
| `maghrib` | `string` | Maghrib time in `HH:MM:SS` (24h)                      | `"19:22:00"`      |
| `isha`    | `string` | Isha/Isyak time in `HH:MM:SS` (24h)                   | `"20:37:00"`      |

**Errors**

| Status | Body | Cause |
|--------|------|-------|
| `404` | `{"message":"No data found for zone: XXXXX for JUNE/2026"}` | No data scraped |

</details>

<details>
<summary><code>GET /api/solat/:zone/:day</code> — Single day prayer time</summary>

```
GET /api/solat/SGR01/15?month=6&year=2026
```

**Query parameters** — same as month endpoint.

**Response** `200` — `prayerTime` is a single object, `periodType` is `"day"`

```json
{
  "prayerTime": {
    "hijri": "1447-12-29",
    "date": "15-Jun-2026",
    "day": "Monday",
    "imsak": "05:30:00",
    "fajr": "05:40:00",
    "syuruk": "07:05:00",
    "dhuha": "07:22:00",
    "dhuhr": "13:15:00",
    "asr": "16:30:00",
    "maghrib": "19:12:00",
    "isha": "20:27:00"
  },
  "status": "OK!",
  "serverTime": "2026-06-26 20:34:11",
  "periodType": "day",
  "lang": "",
  "zone": "SGR01",
  "bearing": ""
}
```
**Fields** — same wrapper and [`prayerTime` fields](#prayer-times-solat) as the month endpoint. `prayerTime` is a **single object** (not an array) and `periodType` is `"day"`.

**Errors**

| Status | Body | Cause |
|--------|------|-------|
| `400` | `{"message":"Invalid day parameter"}` | Non-numeric or out of range |
| `400` | `{"message":"Day 32 out of range for June/2026"}` | Day exceeds month length |
| `404` | Same as month endpoint | No data |

</details>

<details>
<summary><code>GET /api/solat/gps/:lat/:long</code> — Month by GPS (auto-detect zone)</summary>

```
GET /api/solat/gps/3.068498/101.630263?month=6&year=2026
```

**Query parameters** — same as [month endpoint](#prayer-times-solat).

**Response** `200` — same wrapper and [`prayerTime` fields](#prayer-times-solat) as the month endpoint. `zone` reflects the auto-detected JAKIM code.

**Path parameters**

| Param  | Type    | Description | Example    |
|--------|---------|-------------|------------|
| `lat`  | `float` | Latitude    | `3.068498` |
| `long` | `float` | Longitude   | `101.630263` |

**Errors** — same GPS errors as `/api/zones/:lat/:long` (`422`, `404`), plus the standard zone-not-found.

</details>

### Jadual Solat (PDF)

<details>
<summary><code>GET /api/jadual_solat/:zone</code> — Printable PDF prayer timetable</summary>


```
GET /api/jadual_solat/SGR01?month=6&year=2026   # single month
GET /api/jadual_solat/SGR01?year=2026            # full year (12 pages)
```

**Query parameters**

| Param   | Type   | Default | Description                          | Example |
|---------|--------|---------|--------------------------------------|---------|
| `month` | `int`  | (none)  | 1–12; omit to generate all 12 months | `6`     |
| `year`  | `int`  | current | >= 2020                              | `2026`  |

**Response** `200` — `Content-Type: application/pdf`

Landscape A4 PDF. Single-month requests return one page; year-only requests return 12 pages (one per month). Columns: Tarikh, Subuh, Syuruk, Zohor, Asar, Maghrib, Isyak.

**Errors**

| Status | Body | Cause |
|--------|------|-------|
| `404` | `{"message":"No data found for zone: ..."}` | No data |

</details>

### Last Update

The scraper schedule is controlled exclusively via the `SEEDER_SCHED` environment variable — there is no runtime API to enable, disable, or change it.

<details>
<summary><code>GET /api/last-update</code> — Last scraper run info</summary>


```
GET /api/last-update
```

**Response** `200`

```json
{
  "last_run": "2026-06-26T12:53:30Z",
  "last_status": "success: 53/53 zones"
}
```

| Field         | Type     | Description                                                              | Example                    |
|---------------|----------|--------------------------------------------------------------------------|----------------------------|
| `last_run`    | `string` | ISO 8601 timestamp of last scrape, or `""` if never run                  | `"2026-06-26T12:53:30Z"`   |
| `last_status` | `string` | Result of last scrape: `"running"`, `"success: N/N zones"`, `"partial: N/N zones (M failed)"`, or `""` | `"success: 53/53 zones"` |

</details>



### Cache

<details>
<summary><code>POST /api/cache/reset</code> — Invalidate server-side cache</summary>


```
POST /api/cache/reset
X-API-Key: your-secret-key
```

When `API_KEY` is configured, this endpoint requires the `X-API-Key` header. Without a key (or with `API_KEY` left empty), the endpoint is open.

**Response** `200`

```json
{"message": "Cache invalidated"}
```

| Status | Body | Cause |
|--------|------|-------|
| `401` | `{"message":"unauthorized"}` | Missing or wrong `X-API-Key` |

**Per-request invalidation**: Add `?invalidateCache=true` to any cached `GET` request to bypass the cache and re-fetch fresh data for that URL. When `API_KEY` is configured, the `X-API-Key` header is required.

```
GET /api/zones?invalidateCache=true
X-API-Key: your-secret-key
```

</details>

### Fallback

<details>
<summary><code>GET /*</code> — 404 catch-all</summary>

```
GET /api/nonexistent
```

**Response** `404`

```json
{"message": "No route matched. Please see the API documentation."}
```

| Field     | Type     | Description     | Example                                                |
|-----------|----------|-----------------|--------------------------------------------------------|
| `message` | `string` | Error message   | `"No route matched. Please see the API documentation."` |

</details>

## Scraper CLI

```bash
# Seed zones into the database (53 JAKIM zones, embedded in binary)
./scraper seed-zones

# Scrape all zones for a given year (~80s for all 53 zones)
./scraper scrape --year=2026

# Custom DB path
DB_PATH=/tmp/test.db ./scraper seed-zones
DB_PATH=/tmp/test.db ./scraper scrape --year=2025
```

## State / Zone Reference

| Prefix | State | Zones |
|--------|-------|-------|
| JHR | Johor | 01–04 |
| KDH | Kedah | 01–07 |
| KTN | Kelantan | 01–02 |
| MLK | Melaka | 01 |
| NGS | Negeri Sembilan | 01–03 |
| PHG | Pahang | 01–03 |
| PLS | Perlis | 01 |
| PNG | Pulau Pinang | 01 |
| PRK | Perak | 01–04 |
| SBH | Sabah | 01–09 |
| SGR | Selangor | 01–03 |
| SWK | Sarawak | 01–09 |
| TRG | Terengganu | 01–04 |
| WLY | W.P. (Kuala Lumpur, Putrajaya, Labuan) | 01–02 |

## License

MIT

---

Inspired by [mpwt-waktusolat/api-waktusolat-x](https://github.com/mptwaktusolat/api-waktusolat-x).
