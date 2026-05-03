// discord command
package main

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var discordSession *discordgo.Session

func registerDiscordHandlers(s *discordgo.Session) {
	s.AddHandler(onMessageCreate)
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	switch strings.ToLower(m.Content) {
	case "/weather":
		report, err := fetchWeatherReport()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "❌ Cannot fetch weather report.")
			return
		}

		var risk string
		err = db.QueryRow(`SELECT risk_level FROM alerts ORDER BY id DESC LIMIT 1`).Scan(&risk)
		if err != nil {
			risk = "LOW"
		}

		aqi := fetchAQI()
		if aqi != nil {
			sendAQIReport(aqi)
		} else {
			log.Println("/weather: AQI fetch failed")
		}

		sendPeriodicReport(report, risk)
	}
}
