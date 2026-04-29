package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

func sensorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var s SensorData
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		`INSERT INTO sensor_data (sensor_location, humidity, temperature, water_detected) VALUES (?, ?, ?, ?)`,
		s.Location, s.Humidity, s.Temperature, s.WaterDetected,
	)
	if err != nil {
		log.Println("Failed to insert sensor data:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	checkRisk()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func latestAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alert Alert
	err := db.QueryRow(
		`SELECT risk_level, message, timestamp FROM alerts ORDER BY id DESC LIMIT 1`,
	).Scan(&alert.RiskLevel, &alert.Message, &alert.Timestamp)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"risk_level":"LOW","message":"","timestamp":""}`))
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func weatherFetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fetchWeather()
	checkRisk()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// testHighRiskHandler inserts fake high-risk weather+sensor data and fires checkRisk.
// Use this to verify Discord notifications work without waiting for real rain.
func testHighRiskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fake weather: heavy rain probability, west wind
	db.Exec(`INSERT INTO weather_data (temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction)
		VALUES (28.0, 85.0, 80.0, 5.0, 20.0, 270.0)`)

	// Fake sensor: high west-side humidity
	db.Exec(`INSERT INTO sensor_data (sensor_location, humidity, temperature, water_detected)
		VALUES ('west', 90.0, 28.0, 1)`)

	checkRisk()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","note":"Injected HIGH risk data — check Discord"}`))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func testPeroidWeatherHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fetchWeather()

	var weather WeatherData
	err := db.QueryRow(
		`SELECT temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction
		 FROM weather_data ORDER BY id DESC LIMIT 1`,
	).Scan(&weather.Temperature, &weather.Humidity, &weather.RainProbability, &weather.Rainfall, &weather.WindSpeed, &weather.WindDirection)
	if err == sql.ErrNoRows {
		http.Error(w, "No weather data yet", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	s := SensorData{
		Location:    "weather-api",
		Humidity:    weather.Humidity,
		Temperature: weather.Temperature,
	}

	var risk string
	err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&risk)
	if err == sql.ErrNoRows {
		risk = "LOW"
	}

	sendPeriodicReport(weather, s, risk)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","note":"Triggered periodic report — check Discord"}`))
}
