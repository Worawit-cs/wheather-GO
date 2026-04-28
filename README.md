# Home Weather & Flood Alert System

A lightweight IoT backend written in Go that monitors rain and flood risk for your home,
and sends real-time alerts to a Discord channel.

---

## About This Project

This system was built to solve a specific problem: the west side of my house is vulnerable
to heavy rain and flooding when storms come from the west. I needed a way to get an early
warning — before the rain actually arrives — so I have time to prepare.

Instead of building a web dashboard that I would have to remember to open, I chose Discord
as the notification channel. Discord works on my phone, keeps a history of all past alerts,
and sends a push notification automatically. No browser needed.

The system runs on an old machine (Intel Core 2 Duo) and is designed to be as lightweight
as possible: a single compiled Go binary, a SQLite file for storage, and no Docker or
external services required.

---

## What It Does

- Fetches real weather data (rain probability, wind direction, temperature) from the
  Open-Meteo API every 10 minutes — free, no API key required.
- Receives humidity, temperature, and water-detection readings from an ESP32 sensor
  on the west side of the house.
- Runs a risk engine that classifies the current situation as LOW, MEDIUM, or HIGH.
- Sends three types of Discord notifications:
  - **Urgent alert** — fires immediately when risk rises to HIGH, mentions @everyone.
  - **All Clear** — fires when risk drops back to LOW after an elevated state.
  - **Periodic report** — sends a weather summary every 3 hours regardless of risk level.
- Works without an ESP32 connected — falls back to outdoor weather humidity until a
  sensor is available.

---

## Risk Logic

```
West wind = wind direction between 225° and 315°

HIGH   if  rain_probability > 70%
        AND wind is coming from the west
        AND sensor humidity > 80%

MEDIUM if  rain_probability > 50%
        AND sensor humidity > 70%

LOW    otherwise
```

Notifications are only sent when the risk level **changes**, so you will never receive
duplicate alerts for the same condition.

---

## How It Works

```
Open-Meteo API  ──(every 10 min)──>  Go Backend  ──>  SQLite (data.db)
ESP32 Sensor    ──(POST /api/sensor)──>   |
                                          |
                                     Risk Engine
                                          |
                              ┌───────────┴───────────┐
                              v                       v
                       Discord Alert            ESP32 Buzzer
                    (urgent / all clear     (polls GET /api/alert/latest)
                    / 3-hour report)
```

---

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go   | 1.21+   | Install from https://go.dev/dl |
| gcc  | any     | Required to compile the SQLite driver |

**Install gcc on Ubuntu/Debian:**
```bash
sudo apt install build-essential
```

**Install Go on Ubuntu/Debian:**
```bash
sudo apt install golang-go
```

---

## Project Structure

```
wheather-GO/
├── main.go                  # Entry point
├── db.go                    # Database init (SQLite + WAL mode)
├── models.go                # Struct definitions
├── handlers.go              # HTTP handlers
├── risk.go                  # Risk classification engine
├── cron.go                  # Background goroutines (weather check, periodic report)
├── weather_api.go           # Open-Meteo integration
├── notify.go                # Discord webhook notifications
├── simulate.sh              # IoT simulator script (no ESP32 needed)
├── .env                     # Your config (never commit this)
├── .env.example             # Config template (safe to commit)
├── .gitignore
├── go.mod / go.sum
├── weather-server.service   # systemd unit for the server machine
└── deploy.sh                # One-command deploy script
```

---

## Setup

### 1. Clone the repository

```bash
git clone https://github.com/YOUR_USERNAME/wheather-GO.git
cd wheather-GO
```

### 2. Install Go dependencies

```bash
go mod download
```

### 3. Create a Discord Webhook

1. Open Discord and go to the channel where you want alerts.
2. Click the gear icon (Edit Channel) > Integrations > Webhooks.
3. Click **New Webhook**, give it a name (e.g. "Weather Alert"), and copy the URL.

### 4. Find your coordinates

Go to [maps.google.com](https://maps.google.com), right-click your home location, and
click the coordinates at the top of the menu to copy them.

### 5. Create your .env file

```bash
cp .env.example .env
```

Open `.env` and fill in your values:

```env
PORT=3000
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN
LAT=13.7563
LON=100.5018
```

### 6. Build the binary

```bash
go build -o server .
```

### 7. Run

```bash
./server
```

Or run directly without building:

```bash
go run .
```

---

## Testing Without an ESP32

You do not need any hardware to test the system. The simulator script (`simulate.sh`)
sends fake sensor readings to the backend, exactly as an ESP32 would.

**Start the server in one terminal:**
```bash
go run .
```

**Run the simulator in another terminal:**
```bash
# Random humidity readings every 30 seconds
./simulate.sh

# Force HIGH risk conditions (triggers urgent Discord alert)
./simulate.sh high

# Force MEDIUM risk conditions
./simulate.sh medium

# Force LOW risk conditions (triggers All Clear if previous state was elevated)
./simulate.sh low
```

**Change the send interval (default is 30 seconds):**
```bash
INTERVAL=10 ./simulate.sh high
```

**Point the simulator at a different server:**
```bash
SERVER_URL=http://192.168.1.50:3000 ./simulate.sh
```

**Test Discord notifications directly without the simulator:**
```bash
# Inject fake HIGH risk data and fire urgent Discord alert
curl -X POST http://localhost:3000/api/test/high-risk

# Send a low-humidity reading to trigger All Clear
curl -X POST http://localhost:3000/api/sensor \
  -H "Content-Type: application/json" \
  -d '{"location":"west","humidity":45,"temperature":25,"water_detected":0}'
```

---

## Deployment on the Server Machine

### One-time setup

```bash
# 1. Clone the repo
git clone https://github.com/YOUR_USERNAME/wheather-GO.git
cd wheather-GO

# 2. Create .env with your real values
cp .env.example .env
nano .env

# 3. Build the binary
go build -o server .

# 4. Edit the service file — replace 'user' with your actual Linux username
nano weather-server.service

# 5. Install and enable the systemd service
sudo cp weather-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable weather-server
sudo systemctl start weather-server
```

### Deploy an update

After pushing new code to GitHub, run this on the server:

```bash
./deploy.sh
```

This pulls the latest code, rebuilds the binary, and restarts the service automatically.

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET`  | `/health` | Liveness check. Returns `{"status":"ok"}` |
| `POST` | `/api/sensor` | Receive sensor data from ESP32 or simulator |
| `GET`  | `/api/alert/latest` | Get the latest risk level (polled by ESP32 for buzzer) |
| `POST` | `/api/weather/fetch` | Manually trigger a weather fetch from Open-Meteo |
| `POST` | `/api/test/high-risk` | Inject fake HIGH risk data to test Discord alerts |

**POST /api/sensor — request body:**
```json
{
  "location": "west",
  "humidity": 85,
  "temperature": 28,
  "water_detected": 0
}
```

**GET /api/alert/latest — response:**
```json
{
  "risk_level": "HIGH",
  "message": "Risk level changed to HIGH",
  "timestamp": "2026-04-29T10:00:00Z"
}
```

---

## Essential Commands

### Development

```bash
# Run in development mode (auto-reads .env)
go run .

# Build a binary
go build -o server .

# Run the compiled binary
./server

# Download dependencies
go mod download

# Tidy up unused dependencies
go mod tidy
```

### Testing

```bash
# Check server is alive
curl http://localhost:3000/health

# Push a sensor reading
curl -X POST http://localhost:3000/api/sensor \
  -H "Content-Type: application/json" \
  -d '{"location":"west","humidity":85,"temperature":28,"water_detected":0}'

# Get latest alert
curl http://localhost:3000/api/alert/latest

# Manually fetch weather from Open-Meteo
curl -X POST http://localhost:3000/api/weather/fetch

# Fire a test HIGH alert to Discord
curl -X POST http://localhost:3000/api/test/high-risk

# Run the IoT simulator
./simulate.sh high
```

### Systemd Service (Server Machine)

```bash
# Start the service
sudo systemctl start weather-server

# Stop the service
sudo systemctl stop weather-server

# Restart after an update
sudo systemctl restart weather-server

# Check if it is running
sudo systemctl status weather-server

# Enable auto-start on boot
sudo systemctl enable weather-server

# Disable auto-start on boot
sudo systemctl disable weather-server
```

### Logs

```bash
# Watch live server logs
journalctl -u weather-server -f

# Show last 100 lines of logs
journalctl -u weather-server -n 100

# Show logs since a specific time
journalctl -u weather-server --since "2026-04-29 00:00:00"
```

### Database (SQLite)

```bash
# Open the database interactively
sqlite3 data.db

# Inside sqlite3 — view latest weather readings
SELECT * FROM weather_data ORDER BY id DESC LIMIT 10;

# Inside sqlite3 — view latest sensor readings
SELECT * FROM sensor_data ORDER BY id DESC LIMIT 10;

# Inside sqlite3 — view all alerts
SELECT * FROM alerts ORDER BY id DESC LIMIT 20;

# Inside sqlite3 — manually clear all alerts (useful for testing)
DELETE FROM alerts;

# Exit sqlite3
.quit
```

### Git & Deploy

```bash
# Push changes to GitHub (from dev machine)
git add .
git commit -m "your message"
git push

# Deploy to server (run this on the server machine)
./deploy.sh
```

---

## Troubleshooting

**Server will not start — "port already in use"**
```bash
# Find what is using port 3000
sudo lsof -i :3000
# Kill the process (replace PID with the number shown)
kill PID
```

**Server will not start — SQLite compilation error**
```bash
# Install gcc (required by the SQLite Go driver)
sudo apt install build-essential
```

**No Discord messages arriving**
```bash
# Confirm your webhook URL is set
grep DISCORD_WEBHOOK_URL .env

# Test the webhook directly with curl
curl -X POST "$DISCORD_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{"content":"test message"}'
```

**Weather data is not updating**
```bash
# Check that LAT and LON are set in .env
grep -E "LAT|LON" .env

# Manually trigger a fetch and watch the log
curl -X POST http://localhost:3000/api/weather/fetch
```

**Risk never reaches HIGH even with the simulator**

The HIGH threshold requires rain probability > 70% AND west wind AND humidity > 80%.
The weather data fetched from Open-Meteo may show low rain probability if the real
weather is clear. Use the dedicated test endpoint to bypass this:

```bash
curl -X POST http://localhost:3000/api/test/high-risk
```

**systemd service keeps restarting**
```bash
# Check the error in the service log
journalctl -u weather-server -n 50

# Common cause: wrong WorkingDirectory or binary path in the .service file
# Edit the service file, update paths, then reload
sudo nano /etc/systemd/system/weather-server.service
sudo systemctl daemon-reload
sudo systemctl restart weather-server
```

**data.db file is growing too large**

The database stores every sensor and weather reading indefinitely. To clean up old data:
```bash
sqlite3 data.db "DELETE FROM weather_data WHERE timestamp < datetime('now', '-30 days');"
sqlite3 data.db "DELETE FROM sensor_data WHERE timestamp < datetime('-30 days');"
sqlite3 data.db "VACUUM;"
```
