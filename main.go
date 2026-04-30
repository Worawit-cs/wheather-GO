// Package main is a home flood-alert server that monitors weather and air quality
// and sends Discord notifications when rain risk is elevated.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Falls back to real env vars so the same binary works in both dev (.env) and
	// systemd production (env set in the unit file).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	initDB()
	startCron()

	registerRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server running on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
