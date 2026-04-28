package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Open-Meteo API response shape (only fields we use)
type openMeteoResponse struct {
	Current struct {
		Temperature     float64 `json:"temperature_2m"`
		Humidity        float64 `json:"relative_humidity_2m"`
		Rainfall        float64 `json:"rain"`
		WindSpeed       float64 `json:"wind_speed_10m"`
		WindDirection   float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		PrecipitationProbability []float64 `json:"precipitation_probability"`
	} `json:"hourly"`
}

func fetchWeather() {
	lat := os.Getenv("LAT")
	lon := os.Getenv("LON")
	if lat == "" || lon == "" {
		log.Println("LAT/LON not set in .env, skipping weather fetch")
		return
	}

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s"+
			"&current=temperature_2m,relative_humidity_2m,rain,wind_speed_10m,wind_direction_10m"+
			"&hourly=precipitation_probability&forecast_days=1&timezone=auto",
		lat, lon,
	)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Weather API fetch error:", err)
		return
	}
	defer resp.Body.Close()

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("Weather API parse error:", err)
		return
	}

	// Use the next-hour precipitation probability as rain_probability
	rainProb := 0.0
	if len(data.Hourly.PrecipitationProbability) > 0 {
		rainProb = data.Hourly.PrecipitationProbability[0]
	}

	_, err = db.Exec(
		`INSERT INTO weather_data (temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		data.Current.Temperature,
		data.Current.Humidity,
		rainProb,
		data.Current.Rainfall,
		data.Current.WindSpeed,
		data.Current.WindDirection,
	)
	if err != nil {
		log.Println("Failed to insert weather data:", err)
		return
	}

	log.Printf("Weather fetched: rain=%.0f%% wind=%.0f° temp=%.1f°C",
		rainProb, data.Current.WindDirection, data.Current.Temperature)
}
