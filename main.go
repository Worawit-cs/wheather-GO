// Package main is a home flood-alert server that monitors weather and air quality
// and sends Discord notifications when rain risk is elevated.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
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

	// Bot intitialize
	s, err := discordgo.New("Bot " + os.Getenv("WEATHER_BOT_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	registerDiscordHandlers(s)
	s.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentMessageContent

	err = s.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	discordSession = s

	// HTTP server
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
