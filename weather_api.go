package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type openMeteoResponse struct {
	Current struct {
		Time             string  `json:"time"`
		Temperature      float64 `json:"temperature_2m"`
		RelativeHumidity int     `json:"relative_humidity_2m"`
		Precipitation    float64 `json:"precipitation"`
		Rain             float64 `json:"rain"`
		WeatherCode      int     `json:"weather_code"`
		WindSpeed        float64 `json:"wind_speed_10m"`
		WindDirection    float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		Time              []string  `json:"time"`
		Temperature       []float64 `json:"temperature_2m"`
		RelativeHumidity  []int     `json:"relative_humidity_2m"`
		Rain              []float64 `json:"rain"`
		PrecipitationProb []int     `json:"precipitation_probability"`
		WeatherCode       []int     `json:"weather_code"`
		WindSpeed         []float64 `json:"wind_speed_10m"`
	} `json:"hourly"`
}

func wmoCodeText(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45:
		return "Foggy"
	case 48:
		return "Icy fog"
	case 51:
		return "Light drizzle"
	case 53:
		return "Moderate drizzle"
	case 55:
		return "Dense drizzle"
	case 56:
		return "Light freezing drizzle"
	case 57:
		return "Heavy freezing drizzle"
	case 61:
		return "Slight rain"
	case 63:
		return "Moderate rain"
	case 65:
		return "Heavy rain"
	case 66:
		return "Light freezing rain"
	case 67:
		return "Heavy freezing rain"
	case 71:
		return "Slight snowfall"
	case 73:
		return "Moderate snowfall"
	case 75:
		return "Heavy snowfall"
	case 77:
		return "Snow grains"
	case 80:
		return "Slight rain showers"
	case 81:
		return "Moderate rain showers"
	case 82:
		return "Violent rain showers"
	case 85:
		return "Slight snow showers"
	case 86:
		return "Heavy snow showers"
	case 95:
		return "Thunderstorm"
	case 96:
		return "Thunderstorm with slight hail"
	case 99:
		return "Thunderstorm with heavy hail"
	default:
		return fmt.Sprintf("Weather code %d", code)
	}
}

func buildAPIURL(lat, lon string) string {
	return fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s"+
			"&current=temperature_2m,relative_humidity_2m,precipitation,rain,showers,weather_code,wind_speed_10m,wind_direction_10m"+
			"&hourly=temperature_2m,relative_humidity_2m,rain,precipitation_probability,weather_code,wind_speed_10m"+
			"&timezone=Asia%%2FBangkok&past_hours=2&forecast_hours=4",
		lat, lon,
	)
}

func fetchWeatherReport() (*WeatherReport, error) {
	lat := os.Getenv("LAT")
	lon := os.Getenv("LON")
	if lat == "" || lon == "" {
		return nil, fmt.Errorf("LAT/LON not set in .env")
	}

	resp, err := http.Get(buildAPIURL(lat, lon))
	if err != nil {
		return nil, fmt.Errorf("weather API fetch error: %w", err)
	}
	defer resp.Body.Close()

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("weather API parse error: %w", err)
	}

	// current.time uses 15-min precision (e.g. "2026-04-29T15:30") but hourly entries
	// are on the hour ("2026-04-29T15:00"), so truncate to HH:00 before searching.
	currentHour := ""
	if len(data.Current.Time) >= 13 {
		currentHour = data.Current.Time[:13] + ":00"
	}
	currentIdx := -1
	for i, t := range data.Hourly.Time {
		if t == currentHour {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 {
		return nil, fmt.Errorf("could not locate current hour %q (from %q) in hourly array", currentHour, data.Current.Time)
	}

	safeHourly := func(idx int) HourlySnapshot {
		if idx < 0 || idx >= len(data.Hourly.Time) {
			return HourlySnapshot{}
		}
		code := data.Hourly.WeatherCode[idx]
		return HourlySnapshot{
			Time:              data.Hourly.Time[idx],
			Temperature:       data.Hourly.Temperature[idx],
			RelativeHumidity:  data.Hourly.RelativeHumidity[idx],
			Rain:              data.Hourly.Rain[idx],
			PrecipitationProb: data.Hourly.PrecipitationProb[idx],
			WeatherCode:       code,
			WeatherCodeText:   wmoCodeText(code),
			WindSpeed:         data.Hourly.WindSpeed[idx],
		}
	}

	report := &WeatherReport{
		PastHour: safeHourly(currentIdx - 1),
		Current: CurrentWeather{
			Time:             data.Current.Time,
			Temperature:      data.Current.Temperature,
			RelativeHumidity: data.Current.RelativeHumidity,
			Precipitation:    data.Current.Precipitation,
			Rain:             data.Current.Rain,
			WeatherCode:      data.Current.WeatherCode,
			WeatherCodeText:  wmoCodeText(data.Current.WeatherCode),
			WindSpeed:        data.Current.WindSpeed,
			WindDirection:    data.Current.WindDirection,
		},
		Next1Hour:  safeHourly(currentIdx + 1),
		Next2Hours: safeHourly(currentIdx + 2),
		Next3Hours: safeHourly(currentIdx + 3),
	}

	return report, nil
}

// fetchWeather fetches weather, persists current snapshot to DB, and returns the report.
func fetchWeather() *WeatherReport {
	report, err := fetchWeatherReport()
	if err != nil {
		log.Println("fetchWeather error:", err)
		return nil
	}

	c := report.Current
	rainProb := report.Next1Hour.PrecipitationProb

	_, err = db.Exec(
		`INSERT INTO weather_data
			(temperature, humidity, rain_probability, rainfall, wind_speed, wind_direction, weather_code, weather_code_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Temperature,
		c.RelativeHumidity,
		rainProb,
		c.Rain,
		c.WindSpeed,
		c.WindDirection,
		c.WeatherCode,
		c.WeatherCodeText,
	)
	if err != nil {
		log.Println("Failed to insert weather data:", err)
	} else {
		log.Printf("Weather fetched: %s  temp=%.1f°C  rain_prob=%d%%  wind=%.0f°",
			c.WeatherCodeText, c.Temperature, rainProb, c.WindDirection)
	}

	return report
}
