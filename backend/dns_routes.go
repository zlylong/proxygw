package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func registerDNSRoutes(api *gin.RouterGroup) {
	api.GET("/dns", func(c *gin.Context) {
		var local, remote, lazy, mode string
		db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
		db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
		db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazy)
		db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode)
		if strings.TrimSpace(mode) == "" {
			mode = "smart"
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('dns_mode', 'smart')")
		}
		c.JSON(http.StatusOK, gin.H{"local": local, "remote": remote, "lazy": lazy == "true", "mode": mode})
	})

	api.POST("/dns", func(c *gin.Context) {
		var req struct {
			Local, Remote, Mode string
			Lazy                bool
		}
		if c.BindJSON(&req) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		local, ok := normalizeUpstreamCSV(req.Local)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local upstream"})
			return
		}
		remote, ok := normalizeUpstreamCSV(req.Remote)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid remote upstream"})
			return
		}

		mode := strings.TrimSpace(req.Mode)
		if mode == "" {
			mode = "smart"
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_local'", local); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_remote'", remote); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("UPDATE settings SET value=? WHERE key='dns_lazy'", fmt.Sprintf("%t", req.Lazy)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('dns_mode', ?)", mode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		applyMosdnsConfig()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/dns/logs/ws", func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		cmd := exec.Command("tail", "-f", "-n", "20", "/root/proxygw/core/mosdns/mosdns.log")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("failed to read logs"))
			return
		}
		if err := cmd.Start(); err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("failed to start log stream"))
			return
		}
		defer cmd.Process.Kill()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if err := ws.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
				break
			}
		}
	})

	api.GET("/dns/logs", func(c *gin.Context) {
		out, err := exec.Command("tail", "-n", "10", "/root/proxygw/core/mosdns/mosdns.log").Output()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read logs failed"})
			return
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		c.JSON(http.StatusOK, lines)
	})
}
