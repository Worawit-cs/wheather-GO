package main

type SensorData struct {
	Location      string  `json:"location"`
	Humidity      float64 `json:"humidity"`
	Temperature   float64 `json:"temperature"`
	WaterDetected int     `json:"water_detected"`
}

type WeatherData struct {
	Temperature     float64 `json:"temperature"`
	Humidity        float64 `json:"humidity"`
	RainProbability float64 `json:"rain_probability"`
	Rainfall        float64 `json:"rainfall"`
	WindSpeed       float64 `json:"wind_speed"`
	WindDirection   float64 `json:"wind_direction"`
}

type Alert struct {
	RiskLevel string `json:"risk_level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
