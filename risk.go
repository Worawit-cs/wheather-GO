package main

import (
	"database/sql"
	"log"
)

// isWestWind returns true for wind coming from the west (225°–315°)
func isWestWind(degrees float64) bool {
	return degrees >= 225 && degrees <= 315
}

func riskRank(level string) int {
	switch level {
	case "HIGH":
		return 2
	case "MEDIUM":
		return 1
	default:
		return 0
	}
}

func checkRisk() {
	var w WeatherData
	err := db.QueryRow(
		`SELECT temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction
		 FROM weather_data ORDER BY id DESC LIMIT 1`,
	).Scan(&w.Temperature, &w.Humidity, &w.RainProbability, &w.Rainfall, &w.WindSpeed, &w.WindDirection)
	if err == sql.ErrNoRows {
		log.Println("No weather data yet, skipping risk check")
		return
	}
	if err != nil {
		log.Println("Risk check weather query error:", err)
		return
	}

	var s SensorData
	err = db.QueryRow(
		`SELECT sensor_location, humidity, temperature, water_detected
		 FROM sensor_data ORDER BY id DESC LIMIT 1`,
	).Scan(&s.Location, &s.Humidity, &s.Temperature, &s.WaterDetected)
	if err == sql.ErrNoRows {
		// No ESP32 connected yet — fall back to outdoor humidity from weather API
		s.Location = "weather-api"
		s.Humidity = w.Humidity
		s.Temperature = w.Temperature
		log.Println("No sensor data, using weather API humidity as fallback")
	} else if err != nil {
		log.Println("Risk check sensor query error:", err)
		return
	}

	// Classify risk
	newRisk := "LOW"
	if w.RainProbability > 70 && isWestWind(w.WindDirection) && s.Humidity > 80 {
		newRisk = "HIGH"
	} else if w.RainProbability > 50 && s.Humidity > 70 {
		newRisk = "MEDIUM"
	}

	// Read last recorded risk level from DB
	var lastRisk string
	err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&lastRisk)
	if err == sql.ErrNoRows {
		lastRisk = "LOW"
	} else if err != nil {
		log.Println("Risk check alert query error:", err)
		return
	}

	// No change — nothing to do
	if newRisk == lastRisk {
		return
	}

	// Persist the new risk level
	message := "Risk level changed to " + newRisk
	if _, err = db.Exec(`INSERT INTO alerts (risk_level, message) VALUES (?, ?)`, newRisk, message); err != nil {
		log.Println("Failed to insert alert:", err)
		return
	}

	// Notify Discord based on transition
	switch {
	case newRisk == "HIGH":
		sendUrgentAlert(w, s)
	case newRisk == "MEDIUM":
		// Escalation to medium: send as periodic-style embed (no @everyone)
		sendPeriodicReport(w, s, "MEDIUM")
	case newRisk == "LOW" && riskRank(lastRisk) > 0:
		sendAllClear()
	}

	log.Printf("Risk changed: %s → %s", lastRisk, newRisk)
}
