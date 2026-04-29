package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	initDB()
	startCron()

	http.HandleFunc("/api/sensor", sensorHandler)
	http.HandleFunc("/api/alert/latest", latestAlertHandler)
	http.HandleFunc("/api/weather/fetch", weatherFetchHandler)
	http.HandleFunc("/api/weather/report", weatherReportHandler)
	http.HandleFunc("/api/test/high-risk", testHighRiskHandler)
	http.HandleFunc("/api/test/peroid", testPeroidWeatherHandler)
	http.HandleFunc("/health", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server running on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
