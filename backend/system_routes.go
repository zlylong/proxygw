package main

import (
	"log"
	"bufio"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func readCPUUsage() float64 {
	getStat := func() (idle, total float64) {
		b, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0, 0
		}
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "cpu ") {
				fields := strings.Fields(line)
				for i, f := range fields[1:] {
					val, _ := strconv.ParseFloat(f, 64)
					total += val
					if i == 3 {
						idle = val
					}
				}
				return
			}
		}
		return
	}
	idle1, total1 := getStat()
	time.Sleep(200 * time.Millisecond)
	idle2, total2 := getStat()

	if total2-total1 > 0 {
		return 100.0 * (1.0 - (idle2-idle1)/(total2-total1))
	}
	return 0
}

func readMemoryUsage() float64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	memTotal := 0.0
	memAvailable := 0.0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				memTotal, _ = strconv.ParseFloat(parts[1], 64)
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				memAvailable, _ = strconv.ParseFloat(parts[1], 64)
			}
		}
	}
	if memTotal <= 0 {
		return 0
	}
	used := memTotal - memAvailable
	if used < 0 {
		used = 0
	}
	return used / memTotal * 100
}

func registerSystemRoutes(api *gin.RouterGroup) {
	api.GET("/status", func(c *gin.Context) {
		xray := exec.Command("systemctl", "is-active", "--quiet", "xray").Run() == nil
		frr := exec.Command("systemctl", "is-active", "--quiet", "frr").Run() == nil
		mosdns := exec.Command("systemctl", "is-active", "--quiet", "mosdns").Run() == nil

		cpu := readCPUUsage()
		ram := readMemoryUsage()

		var mode string
		if err := db.QueryRow("SELECT value FROM settings WHERE key='mode'").Scan(&mode); err != nil && err != sql.ErrNoRows { log.Printf("[WARN] SELECT value FROM settings WHERE key='mode' err: %v", err) }

		xrayVer := "Unknown"
		xrayVersionOut, err := exec.Command(getPath("core", "xray", "xray"), "version").Output()
		if err == nil {
			xrayVer = parseXrayVersionOutput(string(xrayVersionOut))
		}

		geoVer := "Unknown"
		if data, err := os.ReadFile(getPath("core", "mosdns", "geodata.ver")); err == nil && len(data) > 0 {
			geoVer = strings.TrimSpace(string(data))
		} else if info, err := os.Stat(getPath("core", "mosdns", "geoip.dat")); err == nil {
			geoVer = info.ModTime().Format("2006-01-02")
		}

		upStats, _ := exec.Command(getPath("core", "xray", "xray"), "api", "statsquery", "-server=127.0.0.1:10085", "-name=inbound>>>api_inbound>>>traffic>>>uplink").Output()
		downStats, _ := exec.Command(getPath("core", "xray", "xray"), "api", "statsquery", "-server=127.0.0.1:10085", "-name=inbound>>>api_inbound>>>traffic>>>downlink").Output()
		upStr := "0 MB"
		downStr := "0 MB"
		if strings.Contains(string(upStats), "value") {
			upStr = "Active"
		}
		if strings.Contains(string(downStats), "value") {
			downStr = "Active"
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "running", "mode": mode,
			"xray": xray, "ospf": frr, "mosdns": mosdns,
			"xrayVersion": xrayVer, "geoVersion": geoVer,
			"cpu": fmt.Sprintf("%.1f", cpu), "ram": fmt.Sprintf("%.1f", ram),
			"up": upStr, "down": downStr,
		})
	})

	api.POST("/mode", func(c *gin.Context) {
		var req struct{ Mode string }
		if c.BindJSON(&req) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad mode payload"})
			return
		}
		req.Mode = strings.TrimSpace(req.Mode)
		if req.Mode != "A" && req.Mode != "B" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be A or B"})
			return
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='mode'", req.Mode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if req.Mode == "A" {
			exec.Command("systemctl", "start", "nftables").Run()
			exec.Command("systemctl", "stop", "frr").Run()
		} else {
			exec.Command("systemctl", "start", "nftables").Run()
			exec.Command("systemctl", "start", "frr").Run()
		}
		if err := applyMosdnsConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Mosdns failed: " + err.Error()})
			return
		}
		if err := applyXrayConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Xray failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/cron", func(c *gin.Context) {
		var enabled, cronTime string
		if err := db.QueryRow("SELECT value FROM settings WHERE key='cron_enabled'").Scan(&enabled); err != nil && err != sql.ErrNoRows { log.Printf("[WARN] SELECT value FROM settings WHERE key='cron_enabled' err: %v", err) }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='cron_time'").Scan(&cronTime); err != nil && err != sql.ErrNoRows { log.Printf("[WARN] SELECT value FROM settings WHERE key='cron_time' err: %v", err) }
		if strings.TrimSpace(cronTime) == "" {
			cronTime = "04:00"
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('cron_time', ?)", cronTime)
		}
		c.JSON(http.StatusOK, gin.H{"enabled": enabled == "true", "time": cronTime})
	})

	api.POST("/cron", func(c *gin.Context) {
		var req struct {
			Enabled bool
			Time    string
		}
		if c.BindJSON(&req) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad cron payload"})
			return
		}
		cronTime := strings.TrimSpace(req.Time)
		if cronTime == "" {
			cronTime = "04:00"
		}
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('cron_enabled', ?)", fmt.Sprintf("%t", req.Enabled))
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('cron_time', ?)", cronTime)
		triggerCronReload()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/ospf", func(c *gin.Context) {
		var pub, cand int
		if err := db.QueryRow("SELECT count(*) FROM routes_table WHERE status='published'").Scan(&pub); err != nil && err != sql.ErrNoRows { log.Printf("[WARN] SELECT count(*) FROM routes_table WHERE status='published' err: %v", err) }
		if err := db.QueryRow("SELECT count(*) FROM routes_table WHERE status='candidate'").Scan(&cand); err != nil && err != sql.ErrNoRows { log.Printf("[WARN] SELECT count(*) FROM routes_table WHERE status='candidate' err: %v", err) }

		frrOut, _ := exec.Command("vtysh", "-c", "show ip ospf neighbor json").Output()
		neighborsCount := 0
		if strings.Contains(string(frrOut), "routerId") {
			neighborsCount = 1
		}

		c.JSON(http.StatusOK, gin.H{"neighbors": neighborsCount, "published": pub, "pending": cand, "logs": getOspfLogsSnapshot()})
	})
}
