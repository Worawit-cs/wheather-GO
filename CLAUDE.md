# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run in development (auto-loads .env)
go run .

# Build binary
go build -o wheather-go .

# Simulate IoT sensor data without hardware
./simulate.sh [high|medium|low]

# Deploy to production (pull, rebuild, restart systemd service)
./deploy.sh
```

**No Makefile or test suite exists.** Validate changes manually via HTTP endpoints:
```bash
curl http://localhost:3000/health
curl http://localhost:3000/api/weather/report
curl -X POST http://localhost:3000/api/test/high-risk
curl -X POST http://localhost:3000/api/test/peroid
```

The binary requires `gcc` at build time because `github.com/mattn/go-sqlite3` uses cgo.

## Architecture

Single-package (`main`) flat layout — all `.go` files are in the root directory.

**Data flow:**
1. Background goroutines in `cron.go` run every 10 min (weather check) and every 3 hr (periodic report).
2. Each cycle fetches weather from Open-Meteo (`weather_api.go`) and AQI from WAQI API.
3. `risk.go` classifies the result as HIGH / MEDIUM / LOW based on rain probability and wind direction.
4. `notify.go` sends Discord webhook alerts when risk level changes or on the 3-hour schedule.
5. All data is persisted to SQLite (`data.db`, WAL mode) via helpers in `db.go`.
6. ESP32 sensors POST to `/api/sensor`, but sensor-based risk conditions are currently commented out while awaiting hardware reconnection.

**File responsibilities:**

| File | Role |
|------|------|
| `main.go` | Startup: load `.env`, init DB, start cron goroutines, register HTTP routes |
| `db.go` | SQLite init, schema creation, WAL mode, ALTER TABLE migrations |
| `models.go` | All struct definitions (SensorData, WeatherData, WeatherReport, Alert, AQI) |
| `handlers.go` | HTTP handlers for all routes |
| `risk.go` | Risk classification engine |
| `cron.go` | Two background goroutines (weather check + periodic report) |
| `weather_api.go` | Open-Meteo and WAQI API calls, WMO code-to-text mapping |
| `notify.go` | Discord webhook payload construction and delivery |

## Configuration

Requires a `.env` file in the project root (see `.env.example`):

| Variable | Purpose |
|----------|---------|
| `PORT` | HTTP listen port (default 3000) |
| `DISCORD_WEBHOOK_URL` | Discord webhook for alerts |
| `AQI_TOKEN` | WAQI API token |
| `MAESAI_CODE` | WAQI station code for Mae Sai |
| `LAT` / `LON` | Home coordinates for Open-Meteo |

## Risk Logic

**HIGH**: `rain_probability > 70%` AND wind direction 225°–315° (west wind inflow)  
**MEDIUM**: `rain_probability > 50%`  
**LOW**: everything else

Risk changes trigger immediate Discord notifications; the 3-hour cron always sends a summary regardless of level.

## Production Deployment

The app runs as a systemd service. The unit file is `weather-server.service`. After making changes, `./deploy.sh` handles pull + rebuild + service restart.
