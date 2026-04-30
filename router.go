package main

import "net/http"

// registerRoutes wires URL paths to their handler functions using the standard
// library's default ServeMux. No external router framework is needed at this scale.
func registerRoutes() {
	http.HandleFunc("/api/sensor", sensorHandler)
	http.HandleFunc("/api/alert/latest", latestAlertHandler)
	http.HandleFunc("/api/weather/fetch", weatherFetchHandler)
	http.HandleFunc("/api/weather/report", weatherReportHandler)
	http.HandleFunc("/api/test/high-risk", testHighRiskHandler)
	http.HandleFunc("/api/test/peroid", testPeroidWeatherHandler)
	http.HandleFunc("/api/test/urgent-aqi", testUrgentAQIAlertHandler)
	http.HandleFunc("/health", healthHandler)
}
