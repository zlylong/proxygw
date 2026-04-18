package main

import (
	"bufio"
	"container/ring"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ConnectionRecord struct {
	Time    string `json:"time"`
	Client  string `json:"client"`
	Network string `json:"network"`
	Target  string `json:"target"`
	Policy  string `json:"policy"`
}

var (
	connRing      *ring.Ring
	connRingMutex sync.RWMutex
	// Matches: 2024/04/18 01:23:45 192.168.1.10:4321 accepted tcp:google.com:443 [proxy]
	logRegex = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+from\s+([^\s]+)\s+accepted\s+(tcp|udp):([^\s]+)\s+\[([^\]]+)\]`)
)

func init() {
	connRing = ring.New(200) // Keep last 200 connections
}

func GetRecentConnections() []ConnectionRecord {
	connRingMutex.RLock()
	defer connRingMutex.RUnlock()

	var records []ConnectionRecord
	connRing.Do(func(p interface{}) {
		if p != nil {
			records = append(records, p.(ConnectionRecord))
		}
	})

	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records
}

func StartConnectionTracker() {
	logPath := "/run/proxygw/xray_access.log"

	go func() {
		var file *os.File
		var err error
		var reader *bufio.Reader

		for {
			if file == nil {
				file, err = os.Open(logPath)
				if err != nil {
					time.Sleep(2 * time.Second)
					continue
				}
				file.Seek(0, 2)
				reader = bufio.NewReader(file)
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				stat, errStat := os.Stat(logPath)
				if errStat != nil || stat.Size() == 0 {
					file.Close()
					file = nil
					time.Sleep(1 * time.Second)
					continue
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}

			matches := logRegex.FindStringSubmatch(line)
			if len(matches) == 6 {
				record := ConnectionRecord{
					Time:    matches[1],
					Client:  matches[2],
					Network: matches[3],
					Target:  matches[4],
					Policy:  matches[5],
				}

				if strings.HasPrefix(record.Client, "127.0.0.1") || record.Policy == "api" || record.Policy == "dns-out" {
					continue
				}

				connRingMutex.Lock()
				connRing.Value = record
				connRing = connRing.Next()
				connRingMutex.Unlock()
			}
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Hour)
			stat, err := os.Stat(logPath)
			if err == nil && stat.Size() > 5*1024*1024 {
				os.Remove(logPath)
				exec.Command("systemctl", "kill", "-s", "SIGUSR1", "proxygw-xray").Run()
			}
		}
	}()
}

func registerConnectionRoutes(r *gin.RouterGroup) {
	r.GET("/connections", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    GetRecentConnections(),
		})
	})
}
