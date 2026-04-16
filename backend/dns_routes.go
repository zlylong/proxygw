package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var dnsLogWSConnections int32

func registerDNSRoutes(api *gin.RouterGroup) {
	api.GET("/dns", func(c *gin.Context) {
		var local, remote, lazy, mode string
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazy); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
		if err := db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode); err != nil && err != sql.ErrNoRows { c.JSON(500, gin.H{"error": "db error"}); return }
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
		if err := applyMosdnsConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Mosdns failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.GET("/dns/logs/ws", func(c *gin.Context) {
		const maxDNSLogWSConnections = 16
		if active := atomic.AddInt32(&dnsLogWSConnections, 1); active > maxDNSLogWSConnections {
			atomic.AddInt32(&dnsLogWSConnections, -1)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many log streams"})
			return
		}
		defer atomic.AddInt32(&dnsLogWSConnections, -1)
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		cmd := exec.Command("tail", "-f", "-n", "20", getPath("core", "mosdns", "mosdns.log"))
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("failed to read logs"))
			return
		}
		if err := cmd.Start(); err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("failed to start log stream"))
			return
		}
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			if err := cmd.Wait(); err != nil {
				log.Printf("[WARN] dns log tail process exited: %v", err)
			}
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if err := ws.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[WARN] dns log scanner error: %v", err)
		}
	})

	api.GET("/dns/logs", func(c *gin.Context) {
		out, err := exec.Command("tail", "-n", "10", getPath("core", "mosdns", "mosdns.log")).Output()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read logs failed"})
			return
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		c.JSON(http.StatusOK, lines)
	})
}
