package main

// SensorData is the JSON payload posted by the ESP32 board via /api/sensor.
// WaterDetected is 1 when the water sensor is triggered, 0 otherwise.
type SensorData struct {
	Location      string  `json:"location"`
	Humidity      float64 `json:"humidity"`
	Temperature   float64 `json:"temperature"`
	WaterDetected int     `json:"water_detected"`
}

// WeatherData mirrors the weather_data table row used by checkRisk.
// RainProbability is sourced from the next-hour forecast, not current conditions,
// so it reflects imminent risk rather than what is already falling.
type WeatherData struct {
	Temperature     float64 `json:"temperature"`
	Humidity        float64 `json:"humidity"`
	RainProbability float64 `json:"rain_probability"`
	Rainfall        float64 `json:"rainfall"`
	WindSpeed       float64 `json:"wind_speed"`
	WindDirection   float64 `json:"wind_direction"`
	WeatherCode     int     `json:"weather_code"`
	WeatherCodeText string  `json:"weather_code_text"`
}

// HourlySnapshot represents weather at a single hour (past or forecast).
type HourlySnapshot struct {
	Time              string  `json:"time"`
	Temperature       float64 `json:"temperature"`
	RelativeHumidity  int     `json:"relative_humidity"`
	Rain              float64 `json:"rain"`
	PrecipitationProb int     `json:"precipitation_probability"`
	WeatherCode       int     `json:"weather_code"`
	WeatherCodeText   string  `json:"weather_code_text"`
	WindSpeed         float64 `json:"wind_speed"`
}

// CurrentWeather represents the real-time weather reading from Open-Meteo.
type CurrentWeather struct {
	Time             string  `json:"time"`
	Temperature      float64 `json:"temperature"`
	RelativeHumidity int     `json:"relative_humidity"`
	Precipitation    float64 `json:"precipitation"`
	Rain             float64 `json:"rain"`
	WeatherCode      int     `json:"weather_code"`
	WeatherCodeText  string  `json:"weather_code_text"`
	WindSpeed        float64 `json:"wind_speed"`
	WindDirection    float64 `json:"wind_direction"`
}

// WeatherReport is the full 5-slot report: past 1h, now, next 1/2/3h.
type WeatherReport struct {
	PastHour   HourlySnapshot `json:"past_1h"`
	Current    CurrentWeather `json:"current"`
	Next1Hour  HourlySnapshot `json:"forecast_1h"`
	Next2Hours HourlySnapshot `json:"forecast_2h"`
	Next3Hours HourlySnapshot `json:"forecast_3h"`
}

// Alert records every risk-level transition. The latest row is the current risk state;
// the full history lets you audit when and how often alerts were triggered.
type Alert struct {
	RiskLevel string `json:"risk_level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
