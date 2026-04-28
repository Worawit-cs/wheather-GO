# 🌧️ Home Weather & Flood Alert System (IoT + Web App)

## 🧠 แนวคิดโปรเจกต์

ระบบนี้ถูกออกแบบมาเพื่อ:

* ตรวจจับความเสี่ยงฝนตกหนัก
* วิเคราะห์ว่าฝนจะเข้าทิศที่มีปัญหา (ฝั่งตะวันตกของบ้าน)
* แจ้งเตือนผ่านอุปกรณ์ IoT (ESP32)
* แสดงผลผ่าน Web Dashboard

---

## 🧩 System Overview

```
[Weather API] ---> 
                   \
                    --> [Backend] --> [SQLite DB]
                   /
[ESP32 Sensors] -->

Backend --> Alert --> ESP32 (buzzer)
Backend --> API --> Frontend Dashboard
```

---

## 📡 Inputs

* Weather API (rain probability, wind direction)
* Sensor (humidity, temperature, water detection)

---

## 🧠 Core Logic (Risk Detection)

```
IF
  rain_probability > 70%
  AND wind_direction == WEST
  AND humidity_west > 80
THEN
  ALERT = HIGH
```

---

## 🗄️ Database Schema (SQLite)

### weather_data

```sql
CREATE TABLE weather_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    temperature REAL,
    humidity REAL,
    rain_probability REAL,
    rainfall REAL,
    wind_speed REAL,
    wind_direction TEXT
);
```

### sensor_data

```sql
CREATE TABLE sensor_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    sensor_location TEXT,
    humidity REAL,
    temperature REAL,
    water_detected INTEGER
);
```

### alerts

```sql
CREATE TABLE alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    risk_level TEXT,
    message TEXT
);
```

---

# 🌐 API Design (ใช้ร่วมกันทั้ง Node และ Go)

### POST /api/sensor

```json
{
  "location": "west",
  "humidity": 85,
  "temperature": 28,
  "water_detected": 0
}
```

### POST /api/weather/fetch

ใช้ cron ดึงข้อมูลอากาศ

### GET /api/dashboard

ดูข้อมูลล่าสุด

### GET /api/alert/latest

ให้ ESP32 polling

---

# ⚙️ Implementation: Node.js Version

## install

```
npm init -y
npm install express sqlite3 cors
```

---

## db.js

```js
const sqlite3 = require("sqlite3").verbose();
const db = new sqlite3.Database("./data.db");

db.serialize(() => {
  db.run(`CREATE TABLE IF NOT EXISTS sensor_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    sensor_location TEXT,
    humidity REAL,
    temperature REAL,
    water_detected INTEGER
  )`);

  db.run(`CREATE TABLE IF NOT EXISTS weather_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    rain_probability REAL,
    wind_direction TEXT
  )`);

  db.run(`CREATE TABLE IF NOT EXISTS alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    risk_level TEXT,
    message TEXT
  )`);
});

module.exports = db;
```

---

## server.js

```js
const express = require("express");
const db = require("./db");

const app = express();
app.use(express.json());

function getWeather() {
  return {
    rain_probability: Math.random() * 100,
    wind_direction: "W"
  };
}

function checkRisk(callback) {
  db.get(`SELECT * FROM sensor_data ORDER BY timestamp DESC LIMIT 1`, (err, sensor) => {
    const weather = getWeather();
    let risk = "LOW";

    if (
      weather.rain_probability > 70 &&
      weather.wind_direction === "W" &&
      sensor?.humidity > 80
    ) {
      risk = "HIGH";
    }

    if (risk !== "LOW") {
      db.run(`INSERT INTO alerts (risk_level, message) VALUES (?, ?)`,
        [risk, "Rain risk west side"]);
    }

    callback(risk);
  });
}

app.post("/api/sensor", (req, res) => {
  const { location, humidity, temperature, water_detected } = req.body;

  db.run(
    `INSERT INTO sensor_data (sensor_location, humidity, temperature, water_detected)
     VALUES (?, ?, ?, ?)`,
    [location, humidity, temperature, water_detected],
    () => {
      checkRisk(() => res.json({ status: "ok" }));
    }
  );
});

app.get("/api/alert/latest", (req, res) => {
  db.get(`SELECT * FROM alerts ORDER BY timestamp DESC LIMIT 1`, (err, row) => {
    if (!row) return res.json({ risk: "LOW" });
    res.json(row);
  });
});

app.listen(3000, () => console.log("Server running on 3000"));
```

---

# ⚙️ Implementation: Go Version

## Setup

```bash
go mod init go-weather-alert
go get github.com/mattn/go-sqlite3
```

---

## main.go

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	initDB()
	startCron()

	http.HandleFunc("/api/sensor", sensorHandler)
	http.HandleFunc("/api/alert/latest", latestAlertHandler)

	log.Println("Server running on :3000")
	http.ListenAndServe(":3000", nil)
}
```

---

## db.go

```go
package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		panic(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS sensor_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		location TEXT,
		humidity REAL
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS weather_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		rain_probability REAL,
		wind_direction TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		risk TEXT,
		message TEXT
	)`)
}
```

---

## handlers.go

```go
package main

import (
	"encoding/json"
	"net/http"
)

func sensorHandler(w http.ResponseWriter, r *http.Request) {
	var s Sensor
	json.NewDecoder(r.Body).Decode(&s)

	db.Exec(`INSERT INTO sensor_data (location, humidity) VALUES (?, ?)`,
		s.Location, s.Humidity)

	checkRisk()
	w.Write([]byte(`{"status":"ok"}`))
}

func latestAlertHandler(w http.ResponseWriter, r *http.Request) {
	row := db.QueryRow(`SELECT risk, message FROM alerts ORDER BY timestamp DESC LIMIT 1`)

	var risk, msg string
	err := row.Scan(&risk, &msg)

	if err != nil {
		w.Write([]byte(`{"risk":"LOW"}`))
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"risk": risk,
		"message": msg,
	})
}
```

---

## risk.go

```go
package main

func checkRisk() {
	var humidity float64
	var rain float64
	var wind string

	db.QueryRow(`SELECT humidity FROM sensor_data ORDER BY timestamp DESC LIMIT 1`).Scan(&humidity)
	db.QueryRow(`SELECT rain_probability, wind_direction FROM weather_data ORDER BY timestamp DESC LIMIT 1`).Scan(&rain, &wind)

	risk := "LOW"

	if rain > 70 && wind == "W" && humidity > 80 {
		risk = "HIGH"
	} else if rain > 50 && humidity > 70 {
		risk = "MEDIUM"
	}

	if risk != "LOW" {
		db.Exec(`INSERT INTO alerts (risk, message) VALUES (?, ?)`,
			risk, "Rain coming west side")
	}
}
```

---

## cron.go

```go
package main

import "time"

func startCron() {
	go func() {
		for {
			fetchWeather()
			checkRisk()
			time.Sleep(10 * time.Minute)
		}
	}()
}
```

---

# 🔌 ESP32 Code (ใช้ร่วมกันได้ทั้งสอง backend)

```cpp
#include <WiFi.h>
#include <HTTPClient.h>

const char* ssid = "YOUR_WIFI";
const char* password = "YOUR_PASS";
const char* server = "http://YOUR_SERVER_IP:3000";

int buzzerPin = 13;

void setup() {
  Serial.begin(115200);
  pinMode(buzzerPin, OUTPUT);

  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(1000);
  }
}

float mockHumidity() {
  return random(60, 90);
}

void sendSensor() {
  HTTPClient http;
  http.begin(String(server) + "/api/sensor");
  http.addHeader("Content-Type", "application/json");

  String body = "{\"location\":\"west\",\"humidity\":" + String(mockHumidity()) + ",\"temperature\":28,\"water_detected\":0}";
  http.POST(body);
  http.end();
}

void checkAlert() {
  HTTPClient http;
  http.begin(String(server) + "/api/alert/latest");

  int code = http.GET();
  if (code == 200) {
    String payload = http.getString();
    if (payload.indexOf("HIGH") > 0) {
      tone(buzzerPin, 1000);
      delay(1000);
      noTone(buzzerPin);
    }
  }
  http.end();
}

void loop() {
  sendSensor();
  delay(5000);
  checkAlert();
  delay(5000);
}
```

---

# 🚀 Deployment Notes

* ใช้เครื่องเก่าได้ (ไม่ต้อง Docker)
* SQLite เบามาก
* Go = เบาสุด / Node = dev ง่าย
* ใช้ cron ทุก 5–10 นาที

---

# 💡 Future Improvements

* ต่อ Weather API จริง
* ใช้ sensor จริง (DHT22)
* เพิ่ม dashboard (React / Next.js)
* ทำ graph / trend
* ปรับ rule จาก data จริง

---

# 🎯 Final Summary

* ระบบนี้ “เหมาะกับเครื่องคุณมาก”
* Node = ทำไว
* Go = เสถียร + เบา + เหมาะรันยาว
* โครงสร้างรองรับ IoT + Smart Home จริง

---
