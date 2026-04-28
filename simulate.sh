#!/bin/bash
# Simulates an ESP32 sensor sending data to the backend.
# Usage:
#   ./simulate.sh              — random humidity (60–90%), runs forever
#   ./simulate.sh high         — force HIGH risk (humidity 90%)
#   ./simulate.sh medium       — force MEDIUM risk (humidity 75%)
#   ./simulate.sh low          — force LOW risk (humidity 50%)

SERVER="${SERVER_URL:-http://localhost:3000}"
INTERVAL="${INTERVAL:-30}"   # seconds between readings
MODE="${1:-random}"

echo "Simulator started → $SERVER  (mode: $MODE, interval: ${INTERVAL}s)"
echo "Press Ctrl+C to stop."
echo ""

while true; do
  case "$MODE" in
    high)
      HUMIDITY=90
      TEMP=30
      WATER=1
      ;;
    medium)
      HUMIDITY=75
      TEMP=29
      WATER=0
      ;;
    low)
      HUMIDITY=50
      TEMP=27
      WATER=0
      ;;
    *)
      # Random: humidity between 60 and 90
      HUMIDITY=$(( RANDOM % 31 + 60 ))
      TEMP=$(( RANDOM % 8 + 25 ))
      WATER=$(( RANDOM % 2 ))
      ;;
  esac

  BODY="{\"location\":\"west\",\"humidity\":$HUMIDITY,\"temperature\":$TEMP,\"water_detected\":$WATER}"
  RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER/api/sensor" \
    -H "Content-Type: application/json" \
    -d "$BODY")

  echo "$(date '+%H:%M:%S')  sent → humidity=${HUMIDITY}%  temp=${TEMP}°C  water=${WATER}  → HTTP $RESPONSE"

  sleep "$INTERVAL"
done
