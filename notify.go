package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	colorRed    = 0xFF0000
	colorYellow = 0xFFA500
	colorGreen  = 0x00CC44
)

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color"`
	Fields      []embedField   `json:"fields,omitempty"`
	Footer      *embedFooter   `json:"footer,omitempty"`
	Timestamp   string         `json:"timestamp"`
}

type embedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type embedFooter struct {
	Text string `json:"text"`
}

type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds"`
}

func sendDiscord(payload discordPayload) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("DISCORD_WEBHOOK_URL not set, skipping notification")
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("Discord marshal error:", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Println("Discord webhook error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("Discord webhook returned status %d", resp.StatusCode)
	}
}

func sendUrgentAlert(w WeatherData, s SensorData) {
	embed := discordEmbed{
		Title: "🚨 FLOOD RISK ALERT — HIGH",
		Color: colorRed,
		Fields: []embedField{
			{Name: "🌧️ Rain Probability", Value: fmt.Sprintf("%.0f%%", w.RainProbability), Inline: true},
			{Name: "🌬️ Wind Direction", Value: fmt.Sprintf("%.0f° (West)", w.WindDirection), Inline: true},
			{Name: "💧 Sensor Humidity", Value: fmt.Sprintf("%.0f%%", s.Humidity), Inline: true},
			{Name: "🌡️ Temperature", Value: fmt.Sprintf("%.1f°C", s.Temperature), Inline: true},
			{Name: "💦 Water Detected", Value: waterLabel(s.WaterDetected), Inline: true},
		},
		Footer:    &embedFooter{Text: "West side of house at risk"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendDiscord(discordPayload{
		Content: "@everyone",
		Embeds:  []discordEmbed{embed},
	})
	log.Println("Urgent alert sent to Discord")
}

func sendAllClear() {
	embed := discordEmbed{
		Title:       "✅ All Clear — Risk Resolved",
		Description: "Rain risk has dropped back to LOW. West side is safe.",
		Color:       colorGreen,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	sendDiscord(discordPayload{Embeds: []discordEmbed{embed}})
	log.Println("All Clear sent to Discord")
}

func sendPeriodicReport(w WeatherData, s SensorData, risk string) {
	color := colorGreen
	title := "📊 Weather Report — LOW"
	switch risk {
	case "MEDIUM":
		color = colorYellow
		title = "📊 Weather Report — MEDIUM ⚠️"
	case "HIGH":
		color = colorRed
		title = "📊 Weather Report — HIGH 🚨"
	}

	embed := discordEmbed{
		Title: title,
		Color: color,
		Fields: []embedField{
			{Name: "🌧️ Rain Probability", Value: fmt.Sprintf("%.0f%%", w.RainProbability), Inline: true},
			{Name: "🌬️ Wind Direction", Value: fmt.Sprintf("%.0f°", w.WindDirection), Inline: true},
			{Name: "🌡️ Outdoor Temp", Value: fmt.Sprintf("%.1f°C", w.Temperature), Inline: true},
			{Name: "💧 Sensor Humidity", Value: fmt.Sprintf("%.0f%%", s.Humidity), Inline: true},
			{Name: "🌡️ Sensor Temp", Value: fmt.Sprintf("%.1f°C", s.Temperature), Inline: true},
			{Name: "💦 Water Detected", Value: waterLabel(s.WaterDetected), Inline: true},
		},
		Footer:    &embedFooter{Text: "Next report in 3 hours"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendDiscord(discordPayload{Embeds: []discordEmbed{embed}})
	log.Println("Periodic report sent to Discord")
}

func waterLabel(v int) string {
	if v == 1 {
		return "Yes 💦"
	}
	return "No"
}
