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

	// ESP32 DB write disabled — uncomment when board is connected
	// _, err := db.Exec(
	// 	`INSERT INTO sensor_data (sensor_location, humidity, temperature, water_detected) VALUES (?, ?, ?, ?)`,
	// 	s.Location, s.Humidity, s.Temperature, s.WaterDetected,
	// )
	// if err != nil {
	// 	log.Println("Failed to insert sensor data:", err)
	// 	http.Error(w, "Database error", http.StatusInternalServerError)
	// 	return
	// }
	_ = s // suppress unused variable warning until ESP32 is reconnected

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

// weatherReportHandler returns a fresh WeatherReport JSON (past 1h, now, next 1/2/3h).
func weatherReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	report, err := fetchWeatherReport()
	if err != nil {
		log.Println("weatherReportHandler error:", err)
		http.Error(w, "Failed to fetch weather", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// testHighRiskHandler injects fake high-risk weather data and fires checkRisk.
func testHighRiskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fake weather: heavy rain probability, west wind
	db.Exec(`INSERT INTO weather_data (temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction, weather_code, weather_code_text)
		VALUES (28.0, 85.0, 80.0, 5.0, 20.0, 270.0, 63, 'Moderate rain')`)

	// Mock sensor INSERT disabled — uncomment when ESP32 is connected
	// db.Exec(`INSERT INTO sensor_data (sensor_location, humidity, temperature, water_detected)
	// 	VALUES ('west', 90.0, 28.0, 1)`)

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

	report, err := fetchWeatherReport()
	if err != nil {
		log.Println("testPeroidWeatherHandler error:", err)
		http.Error(w, "Failed to fetch weather", http.StatusInternalServerError)
		return
	}

	var risk string
	err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&risk)
	if err == sql.ErrNoRows {
		risk = "LOW"
	}

	sendPeriodicReport(report, risk)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","note":"Triggered periodic report — check Discord"}`))
}
