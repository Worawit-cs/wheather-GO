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

	// ESP32 sensor query — uncomment when board is connected
	// var s SensorData
	// err = db.QueryRow(
	// 	`SELECT sensor_location, humidity, temperature, water_detected
	// 	 FROM sensor_data ORDER BY id DESC LIMIT 1`,
	// ).Scan(&s.Location, &s.Humidity, &s.Temperature, &s.WaterDetected)
	// if err != nil && err != sql.ErrNoRows {
	// 	log.Println("Risk check sensor query error:", err)
	// 	return
	// }

	// Risk classification based on weather data only (no sensor required)
	newRisk := "LOW"
	if w.RainProbability > 70 && isWestWind(w.WindDirection) {
		newRisk = "HIGH"
	} else if w.RainProbability > 50 {
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
		report, fetchErr := fetchWeatherReport()
		if fetchErr != nil {
			log.Println("Could not fetch report for urgent alert:", fetchErr)
		} else {
			sendUrgentAlert(report)
		}
	case newRisk == "MEDIUM":
		report, fetchErr := fetchWeatherReport()
		if fetchErr != nil {
			log.Println("Could not fetch report for medium alert:", fetchErr)
		} else {
			sendPeriodicReport(report, "MEDIUM")
		}
	case newRisk == "LOW" && riskRank(lastRisk) > 0:
		sendAllClear()
	}

	log.Printf("Risk changed: %s → %s", lastRisk, newRisk)
}
