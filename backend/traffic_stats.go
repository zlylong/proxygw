package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type XrayStat struct {
	Stat []struct {
		Name  string `json:"name"`
		Value int64  `json:"value"`
	} `json:"stat"`
}

var (
	trafficMutex sync.Mutex
	currentSpeedUp   int64
	currentSpeedDown int64
	
	lastTotalUp   int64
	lastTotalDown int64
)

func initTrafficDB() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS traffic_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts DATETIME DEFAULT CURRENT_TIMESTAMP,
		up_bytes INTEGER,
		down_bytes INTEGER
	)`)
	if err != nil {
		log.Printf("[WARN] Failed to create traffic_history table: %v", err)
	}
	db.Exec(`DELETE FROM traffic_history WHERE ts < datetime('now', '-24 hours')`)
}

func startTrafficMonitor() {
	initTrafficDB()
	
	ticker := time.NewTicker(2 * time.Second)
	saveTicker := time.NewTicker(5 * time.Minute)
	
	var accumUp, accumDown int64
	
	for {
		select {
		case <-ticker.C:
			out, err := exec.Command(getPath("core", "xray", "xray"), "api", "statsquery", "-server=127.0.0.1:10085", "-pattern=").Output()
			if err != nil {
				trafficMutex.Lock()
				currentSpeedUp = 0
				currentSpeedDown = 0
				lastTotalUp = 0
				lastTotalDown = 0
				trafficMutex.Unlock()
				continue
			}
			
			var stats XrayStat
			if err := json.Unmarshal(out, &stats); err != nil {
				continue
			}
			
			var totalUp, totalDown int64
			for _, s := range stats.Stat {
				if strings.Contains(s.Name, ">>>uplink") && !strings.Contains(s.Name, "api_inbound") {
					totalUp += s.Value
				}
				if strings.Contains(s.Name, ">>>downlink") && !strings.Contains(s.Name, "api_inbound") {
					totalDown += s.Value
				}
			}
			
			trafficMutex.Lock()
			deltaUp := totalUp - lastTotalUp
			deltaDown := totalDown - lastTotalDown
			
			if lastTotalUp == 0 || deltaUp < 0 { deltaUp = 0 }
			if lastTotalDown == 0 || deltaDown < 0 { deltaDown = 0 }
			
			currentSpeedUp = deltaUp / 2
			currentSpeedDown = deltaDown / 2
			
			lastTotalUp = totalUp
			lastTotalDown = totalDown
			
			accumUp += deltaUp
			accumDown += deltaDown
			trafficMutex.Unlock()
			
		case <-saveTicker.C:
			trafficMutex.Lock()
			saveUp := accumUp
			saveDown := accumDown
			accumUp = 0
			accumDown = 0
			trafficMutex.Unlock()
			
			if saveUp > 0 || saveDown > 0 {
				db.Exec(`INSERT INTO traffic_history (up_bytes, down_bytes) VALUES (?, ?)`, saveUp, saveDown)
			}
			db.Exec(`DELETE FROM traffic_history WHERE ts < datetime('now', '-24 hours')`)
		}
	}
}
