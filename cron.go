package main

import (
	"database/sql"
	"log"
	"time"
)

func startCron() {
	// Goroutine 1: fetch weather + AQI + check risk every 10 minutes
	go func() {
		for {
			fetchWeather()
			checkRisk()
			time.Sleep(10 * time.Minute)
		}
	}()

	// Goroutine 2: send periodic Discord report every 3 hours
	go func() {
		for {
			report, err := fetchWeatherReport()
			if err != nil {
				log.Println("Periodic report skipped: could not fetch weather:", err)
				continue
			}

			// ESP32 sensor query — uncomment when board is connected
			// var s SensorData
			// err = db.QueryRow(
			// 	`SELECT sensor_location, humidity, temperature, water_detected
			// 	 FROM sensor_data ORDER BY id DESC LIMIT 1`,
			// ).Scan(&s.Location, &s.Humidity, &s.Temperature, &s.WaterDetected)
			// if err != nil && err != sql.ErrNoRows {
			// 	log.Println("Periodic report sensor query error:", err)
			// }

			var risk string
			err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&risk)
			if err == sql.ErrNoRows {
				risk = "LOW"
			}

			aqi := fetchAQI()
			if aqi != nil {
				sendAQIReport(aqi)
			} else {
				log.Println("Periodic AQI report skipped: could not fetch AQI")
			}

			sendPeriodicReport(report, risk)
			time.Sleep(3 * time.Hour)
		}
	}()

	log.Println("Cron started: weather check every 10min, report every 3hr")
}

// keep sql import used by latestAlertHandler via db.QueryRow
var _ = sql.ErrNoRows
