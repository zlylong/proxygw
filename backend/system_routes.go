package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerSystemRoutes(api *gin.RouterGroup) {
	api.GET("/status", func(c *gin.Context) {
		xray := exec.Command("systemctl", "is-active", "--quiet", "xray").Run() == nil
		frr := exec.Command("systemctl", "is-active", "--quiet", "frr").Run() == nil
		mosdns := exec.Command("systemctl", "is-active", "--quiet", "mosdns").Run() == nil

		cpuOut, _ := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print $2}'").Output()
		cpu, _ := strconv.ParseFloat(strings.TrimSpace(string(cpuOut)), 64)
		ramOut, _ := exec.Command("sh", "-c", "free | grep Mem | awk '{print $3/$2 * 100.0}'").Output()
		ram, _ := strconv.ParseFloat(strings.TrimSpace(string(ramOut)), 64)

		var mode string
		db.QueryRow("SELECT value FROM settings WHERE key='mode'").Scan(&mode)

		xrayVer := "Unknown"
		xrayVersionOut, err := exec.Command("xray", "version").Output()
		if err == nil {
			xrayVer = parseXrayVersionOutput(string(xrayVersionOut))
		}

		geoVer := "Unknown"
		if data, err := os.ReadFile("/usr/local/bin/geodata.ver"); err == nil && len(data) > 0 {
			geoVer = strings.TrimSpace(string(data))
		} else if info, err := os.Stat("/usr/local/bin/geoip.dat"); err == nil {
			geoVer = info.ModTime().Format("2006-01-02")
		}

		upStats, _ := exec.Command("xray", "api", "statsquery", "-server=127.0.0.1:10085", "-name=inbound>>>api_inbound>>>traffic>>>uplink").Output()
		downStats, _ := exec.Command("xray", "api", "statsquery", "-server=127.0.0.1:10085", "-name=inbound>>>api_inbound>>>traffic>>>downlink").Output()
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
			exec.Command("systemctl", "stop", "nftables").Run()
			exec.Command("systemctl", "start", "frr").Run()
		}
		applyMosdnsConfig()
		applyXrayConfig()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/cron", func(c *gin.Context) {
		var enabled, cronTime string
		db.QueryRow("SELECT value FROM settings WHERE key='cron_enabled'").Scan(&enabled)
		db.QueryRow("SELECT value FROM settings WHERE key='cron_time'").Scan(&cronTime)
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
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/ospf", func(c *gin.Context) {
		var pub, cand int
		db.QueryRow("SELECT count(*) FROM routes_table WHERE status='published'").Scan(&pub)
		db.QueryRow("SELECT count(*) FROM routes_table WHERE status='candidate'").Scan(&cand)

		frrOut, _ := exec.Command("vtysh", "-c", "show ip ospf neighbor json").Output()
		neighborsCount := 0
		if strings.Contains(string(frrOut), "routerId") {
			neighborsCount = 1
		}

		c.JSON(http.StatusOK, gin.H{"neighbors": neighborsCount, "published": pub, "pending": cand, "logs": getOspfLogsSnapshot()})
	})
}
