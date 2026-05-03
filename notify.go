package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	colorRed    = 0xFF0000
	colorYellow = 0xFFA500
	colorGreen  = 0x00CC44
)

// Testing BOT
const DEBUG = true

type discordEmbed struct {
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color"`
	Fields      []embedField `json:"fields,omitempty"`
	Footer      *embedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp"`
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

// function code-text mapping with their colour
func codeToColour(code string) int {
	switch strings.ToLower(code) {
	case "good":
		return 5857280
	case "moderate":
		return 16776960
	case "unhealthy for sensitive groups":
		return 16744448
	case "unhealthy":
		return 16711680
	case "very unhealthy":
		return 9381719
	case "hazardous":
		return 8257539
	default:
		return 8421504 // เทา fallback
	}
}

func toDiscordgoEmbed(e discordEmbed) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       e.Title,
		Description: e.Description,
		Color:       e.Color,
		Timestamp:   e.Timestamp,
	}
	for _, f := range e.Fields {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   f.Name,
			Value:  f.Value,
			Inline: f.Inline,
		})
	}
	if e.Footer != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: e.Footer.Text}
	}
	return embed
}

func sendBotToMaesai(payload discordPayload) {
	channelID := os.Getenv("MAESAI_CHANNEL")
	if discordSession == nil || channelID == "" {
		return
	}
	embeds := make([]*discordgo.MessageEmbed, 0, len(payload.Embeds))
	for _, e := range payload.Embeds {
		embeds = append(embeds, toDiscordgoEmbed(e))
	}
	_, err := discordSession.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: payload.Content,
		Embeds:  embeds,
	})
	if err != nil {
		log.Printf("Bot channel send error: %v", err)
	}
}

// sendDiscord is the single exit point for all Discord notifications.
// All other send* functions build a payload and call this.
func sendDiscord(payload discordPayload) {
	URL := "DISCORD_WEBHOOK_MAESAI_URL"
	if DEBUG {
		URL = os.Getenv("DISCORD_WEBHOOK_TEST_URL")
	}
	webhookURL := os.Getenv(URL)
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

func hourlyField(label, t string, snap HourlySnapshot) embedField {
	return embedField{
		Name: label + " (" + shortTime(t) + ")",
		Value: fmt.Sprintf(
			"🌡️ %.1f°C  💧 %d%%  🌬️ %.1f km/h\n🌧️ Rain: %.1fmm  📊 Prob: %d%%  ☁️ %s",
			snap.Temperature, snap.RelativeHumidity, snap.WindSpeed,
			snap.Rain, snap.PrecipitationProb, snap.WeatherCodeText,
		),
		Inline: false,
	}
}

func forecastField(label, t string, snap HourlySnapshot) embedField {
	return embedField{
		Name: label + " (" + shortTime(t) + ")",
		Value: fmt.Sprintf(
			"🌡️ %.1f°C  📊 Rain prob: %d%%  ☁️ %s",
			snap.Temperature, snap.PrecipitationProb, snap.WeatherCodeText,
		),
		Inline: false,
	}
}

func shortTime(t string) string {
	// t is "2006-01-02T15:04" — extract HH:MM
	if len(t) >= 16 {
		return t[11:16]
	}
	return t
}

// sendUrgentAlert fires when risk transitions to HIGH. Uses @everyone so the
// notification breaks through Do Not Disturb on mobile Discord.
func sendUrgentWeatherAlert(report *WeatherReport) {
	c := report.Current
	embed := discordEmbed{
		Title: "🚨 FLOOD RISK ALERT — HIGH",
		Color: colorRed,
		Fields: []embedField{
			{Name: "🌧️ Rain Probability (next 1h)", Value: fmt.Sprintf("%d%%", report.Next1Hour.PrecipitationProb), Inline: true},
			{Name: "🌬️ Wind Direction", Value: fmt.Sprintf("%.0f°", c.WindDirection), Inline: true},
			{Name: "💨 Wind Speed", Value: fmt.Sprintf("%.1f km/h", c.WindSpeed), Inline: true},
			{Name: "🌡️ Temperature", Value: fmt.Sprintf("%.1f°C", c.Temperature), Inline: true},
			{Name: "💧 Humidity", Value: fmt.Sprintf("%d%%", c.RelativeHumidity), Inline: true},
			{Name: "☁️ Condition", Value: c.WeatherCodeText, Inline: true},
			{Name: "🔬 Sensor", Value: "Location: -  |  Humidity: -  |  Temp: -  |  Water: -", Inline: false},
		},
		Footer:    &embedFooter{Text: "West side of house at risk"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	p := discordPayload{Content: "@everyone", Embeds: []discordEmbed{embed}}
	sendDiscord(p)
	sendBotToMaesai(p)
	log.Println("Urgent alert sent to Discord (weather report)")
}

// send urgent aqi alert when AQI is overwelhm
func sendUrgentAQIAlert(aqi *aqiResponse) {
	c := aqi.CurrentAQI
	embed := discordEmbed{
		Title: fmt.Sprintf("🚨 AQI RISK ALERT — %s 💨\n ⚠️status: %s", c.City, c.CodeText),
		Color: codeToColour(c.CodeText),
		Fields: []embedField{
			{
				Name:   "🕒 TIME",
				Value:  fmt.Sprintf("%s", shortTime(c.Time)),
				Inline: true,
			},
			{
				Name:   "📊 AQI",
				Value:  fmt.Sprintf("%d", c.AQI),
				Inline: true,
			},
			{
				Name:   "💨 PM2.5",
				Value:  fmt.Sprintf("%.1f μg/m³", c.PM25),
				Inline: true,
			},
			{
				Name:   "🌫️ PM10",
				Value:  fmt.Sprintf("%.1f μg/m³", c.PM10),
				Inline: true,
			},
		},
		Footer:    &embedFooter{Text: "Avoid outdoor activities ⚠️"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	p := discordPayload{Content: "@everyone", Embeds: []discordEmbed{embed}}
	sendDiscord(p)
	sendBotToMaesai(p)
	log.Println("Urgent alert sent to Discord (weather report)")
}

func sendAllClear() {
	embed := discordEmbed{
		Title:       "✅ All Clear — Risk Resolved",
		Description: "Rain risk has dropped back to LOW. West side is safe.",
		Color:       colorGreen,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	p := discordPayload{Embeds: []discordEmbed{embed}}
	sendDiscord(p)
	sendBotToMaesai(p)
	log.Println("All Clear sent to Discord")
}

// aqiReportLabel maps an AQI value + WAQI code text to the correct
// Discord embed title and colour for the periodic AQI report.
func aqiReportLabel(aqiVal int, codeText string) (string, int) {
	var title string
	switch {
	case aqiVal > 300:
		title = "🌫️ Air Quality Report — HAZARDOUS"
	case aqiVal > 200:
		title = "🌫️ Air Quality Report — VERY UNHEALTHY"
	case aqiVal > 150:
		title = "🌫️ Air Quality Report — UNHEALTHY"
	case aqiVal > 100:
		title = "🌫️ Air Quality Report — UNHEALTHY FOR SENSITIVE GROUPS"
	case aqiVal > 50:
		title = "🌫️ Air Quality Report — MODERATE"
	default:
		title = "🌫️ Air Quality Report — GOOD"
	}
	return title, codeToColour(codeText)
}

func sendAQIReport(aqi *aqiResponse) {
	c := aqi.CurrentAQI
	title, color := aqiReportLabel(c.AQI, c.CodeText)

	// Derive "today" from the WAQI response timestamp (station's local timezone),
	// not time.Now(), to avoid a server UTC vs. station UTC+7 mismatch.
	aqiDateStr := c.Time
	if len(aqiDateStr) >= 10 {
		aqiDateStr = aqiDateStr[:10]
	}
	todayT, err := time.Parse("2006-01-02", aqiDateStr)
	if err != nil {
		todayT = time.Now()
	}
	today := todayT.Format("2006-01-02")
	yesterday := todayT.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := todayT.AddDate(0, 0, 1).Format("2006-01-02")

	findPM10 := func(day string) *PM10Detail {
		for i := range aqi.DailyAQI.PM10 {
			if aqi.DailyAQI.PM10[i].Day == day {
				return &aqi.DailyAQI.PM10[i]
			}
		}
		return nil
	}
	findPM25 := func(day string) *PM25Detail {
		for i := range aqi.DailyAQI.PM25 {
			if aqi.DailyAQI.PM25[i].Day == day {
				return &aqi.DailyAQI.PM25[i]
			}
		}
		return nil
	}
	fmtPM10 := func(d *PM10Detail) string {
		if d == nil {
			return "—"
		}
		return fmt.Sprintf("⬇️ %d   ↔️ %d   ⬆️ %d", d.Min, d.Avg, d.Max)
	}
	fmtPM25 := func(d *PM25Detail) string {
		if d == nil {
			return "—"
		}
		return fmt.Sprintf("⬇️ %d   ↔️ %d   ⬆️ %d", d.Min, d.Avg, d.Max)
	}

	embed := discordEmbed{
		Title: title,
		Color: color,
		Fields: []embedField{
			{
				Name: "📍 City / Time",
				Value: fmt.Sprintf("%s\n🕐 %s",
					c.City, shortTime(c.Time),
				),
				Inline: false,
			},
			{
				Name: "📊 Now",
				Value: fmt.Sprintf(
					"💨 PM10:   %.1f μg/m³\n🌫️ PM2.5:   %.1f μg/m³\nAQI: %d — %s",
					c.PM10, c.PM25, c.AQI, aqiCodeText(c.AQI),
				),
				Inline: false,
			},
			{
				Name: "📅 Daily Forecast",
				Value: fmt.Sprintf(
					"💨 PM10\n  ⏪ Yesterday   %s\n  📍 Today       %s\n  🔮 Tomorrow    %s\n\n🌫️ PM2.5\n  ⏪ Yesterday   %s\n  📍 Today       %s\n  🔮 Tomorrow    %s",
					fmtPM10(findPM10(yesterday)), fmtPM10(findPM10(today)), fmtPM10(findPM10(tomorrow)),
					fmtPM25(findPM25(yesterday)), fmtPM25(findPM25(today)), fmtPM25(findPM25(tomorrow)),
				),
				Inline: false,
			},
		},
		Footer:    &embedFooter{Text: "Mae Sai AQI — updated every 3 hours"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	p := discordPayload{Embeds: []discordEmbed{embed}}
	sendDiscord(p)
	sendBotToMaesai(p)
	log.Println("AQI report sent to Discord")
}

func sendPeriodicReport(report *WeatherReport, risk string) {
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

	c := report.Current
	fields := []embedField{
		hourlyField("⏪ 1 Hour Ago", report.PastHour.Time, report.PastHour),
		{
			Name: "📍 Now (" + shortTime(c.Time) + ")",
			Value: fmt.Sprintf(
				"🌡️ %.1f°C  💧 %d%%  🌬️ %.1f km/h (%.0f°)\n🌧️ Rain: %.1fmm  ☁️ %s",
				c.Temperature, c.RelativeHumidity, c.WindSpeed, c.WindDirection,
				c.Rain, c.WeatherCodeText,
			),
			Inline: false,
		},
		forecastField("🔮 +1h", report.Next1Hour.Time, report.Next1Hour),
		forecastField("🔮 +2h", report.Next2Hours.Time, report.Next2Hours),
		forecastField("🔮 +3h", report.Next3Hours.Time, report.Next3Hours),
		{Name: "🔬 Sensor", Value: "Location: -  |  Humidity: -  |  Temp: -  |  Water: -", Inline: false},
	}

	embed := discordEmbed{
		Title:     title,
		Color:     color,
		Fields:    fields,
		Footer:    &embedFooter{Text: "Next report in 3 hours"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	p := discordPayload{Embeds: []discordEmbed{embed}}
	sendDiscord(p)
	sendBotToMaesai(p)
	log.Println("Periodic report sent to Discord")
}
