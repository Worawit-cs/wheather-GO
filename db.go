package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// WAL mode: safe for concurrent cron goroutine + HTTP handlers
	if _, err = db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Fatal("Failed to set WAL mode:", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS weather_data (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp        DATETIME DEFAULT CURRENT_TIMESTAMP,
			temperature      REAL,
			humidity         REAL,
			rain_probability REAL,
			rainfall         REAL,
			wind_speed       REAL,
			wind_direction   REAL,
			weather_code     INTEGER DEFAULT 0,
			weather_code_text TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS sensor_data (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp        DATETIME DEFAULT CURRENT_TIMESTAMP,
			sensor_location  TEXT,
			humidity         REAL,
			temperature      REAL,
			water_detected   INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp        DATETIME DEFAULT CURRENT_TIMESTAMP,
			risk_level       TEXT,
			message          TEXT
		)`,
	}

	for _, stmt := range statements {
		if _, err = db.Exec(stmt); err != nil {
			log.Fatal("Failed to create table:", err)
		}
	}

	// Migrations for existing DB — errors ignored when column already exists
	db.Exec(`ALTER TABLE weather_data ADD COLUMN weather_code INTEGER DEFAULT 0`)
	db.Exec(`ALTER TABLE weather_data ADD COLUMN weather_code_text TEXT DEFAULT ''`)

	log.Println("Database initialized")
}
