package main

import (
	"database/sql"
	"log"
	"time"
)

func startCron() {
	// Goroutine 1: fetch weather + check risk every 10 minutes
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
			time.Sleep(3 * time.Hour)

			var w WeatherData
			err := db.QueryRow(
				`SELECT temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction
				 FROM weather_data ORDER BY id DESC LIMIT 1`,
			).Scan(&w.Temperature, &w.Humidity, &w.RainProbability, &w.Rainfall, &w.WindSpeed, &w.WindDirection)
			if err == sql.ErrNoRows {
				log.Println("Periodic report skipped: no weather data yet")
				continue
			}

			var s SensorData
			err = db.QueryRow(
				`SELECT sensor_location, humidity, temperature, water_detected
				 FROM sensor_data ORDER BY id DESC LIMIT 1`,
			).Scan(&s.Location, &s.Humidity, &s.Temperature, &s.WaterDetected)
			if err == sql.ErrNoRows {
				// No ESP32 yet — use weather API values as fallback
				s.Location = "weather-api"
				s.Humidity = w.Humidity
				s.Temperature = w.Temperature
			}

			var risk string
			err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&risk)
			if err == sql.ErrNoRows {
				risk = "LOW"
			}

			sendPeriodicReport(w, s, risk)
		}
	}()

	log.Println("Cron started: weather check every 10min, report every 3hr")
}
