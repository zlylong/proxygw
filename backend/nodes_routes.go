package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func registerNodeRoutes(api *gin.RouterGroup) {
	api.GET("/nodes", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, name, grp, type, address, port, uuid, active, ping, COALESCE(params, '{}') FROM nodes")
		defer rows.Close()
		var nodes []map[string]interface{}
		for rows.Next() {
			var id, port, ping int
			var name, grp, ntype, address, uuid, params string
			var active bool
			rows.Scan(&id, &name, &grp, &ntype, &address, &port, &uuid, &active, &ping, &params)
			nodes = append(nodes, map[string]interface{}{"id": id, "name": name, "group": grp, "type": ntype, "address": address, "port": port, "uuid": uuid, "active": active, "ping": ping, "params": params})
		}
		if nodes == nil {
			nodes = make([]map[string]interface{}, 0)
		}
		c.JSON(http.StatusOK, nodes)
	})

	api.POST("/nodes", func(c *gin.Context) {
		var n struct {
			Name, Group, Type, Address, UUID string
			Port                             int
		}
		if c.BindJSON(&n) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		if _, err := db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, ?, ?, ?, ?, ?, '{}', 1)", n.Name, n.Group, n.Type, n.Address, n.Port, n.UUID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.POST("/nodes/import", func(c *gin.Context) {
		var req struct{ Url string }
		if err := c.BindJSON(&req); err == nil {
			if strings.HasPrefix(req.Url, "vmess://") {
				b64 := strings.TrimPrefix(req.Url, "vmess://")
				decoded, _ := base64.StdEncoding.DecodeString(b64)
				var v struct {
					Ps, Add, Id string
					Port        interface{}
				}
				json.Unmarshal(decoded, &v)
				portInt := parsePortValue(v.Port)
				
				vmessSettings := map[string]interface{}{
				    "vnext": []map[string]interface{}{{"users": []map[string]interface{}{{"id": v.Id, "alterId": 0}}}},
				}
				finalParamsVmess := map[string]interface{}{"settings": vmessSettings}
				vmessParamsJson, _ := json.Marshal(finalParamsVmess)
				db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, 'Imported', 'Vmess', ?, ?, '', ?, 1)", v.Ps, v.Add, portInt, string(vmessParamsJson))

				scheduleApply()
				c.JSON(http.StatusOK, gin.H{"success": true})
				return
			} else if strings.HasPrefix(req.Url, "vless://") {
				parsedUrl, err := url.Parse(req.Url)
				if err == nil {
					uuid := parsedUrl.User.Username()
					host := parsedUrl.Hostname()
					portStr := parsedUrl.Port()
					portInt, _ := strconv.Atoi(portStr)
					alias, _ := url.QueryUnescape(parsedUrl.Fragment)
					if alias == "" {
						alias = host
					}

					query := parsedUrl.Query()
					params := map[string]interface{}{
						"encryption": query.Get("encryption"),
						"flow":       query.Get("flow"),
						"security":   query.Get("security"),
						"sni":        query.Get("sni"),
						"fp":         query.Get("fp"),
						"pbk":        query.Get("pbk"),
						"sid":        query.Get("sid"),
						"type":       query.Get("type"),
						"headerType": query.Get("headerType"),
					}
					
					// Convert immediately to Xray standard structure
					ss := map[string]interface{}{"network": params["type"]}
					if params["security"] != nil && params["security"] != "" {
						ss["security"] = params["security"]
					}
					if params["security"] == "reality" {
						ss["realitySettings"] = map[string]interface{}{
							"fingerprint": params["fp"], "serverName": params["sni"],
							"publicKey": params["pbk"], "shortId": params["sid"], "spiderX": "/",
						}
					} else if params["security"] == "tls" {
						ss["tlsSettings"] = map[string]interface{}{"serverName": params["sni"]}
					}
					
					settings := map[string]interface{}{
					    "vnext": []map[string]interface{}{{"users": []map[string]interface{}{{"id": uuid, "encryption": "none"}}}},
					}
					
					finalParams := map[string]interface{}{
					    "settings": settings,
					    "streamSettings": ss,
					}
					
					paramsJson, _ := json.Marshal(finalParams)

					db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, 'Imported', 'Vless', ?, ?, '', ?, 1)", alias, host, portInt, string(paramsJson))
					scheduleApply()
					c.JSON(http.StatusOK, gin.H{"success": true})
					return
				}
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported URL format"})
	})

	api.POST("/nodes/ping", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, address, port FROM nodes")
		defer rows.Close()
		const maxConcurrentPing = 20
		sem := make(chan struct{}, maxConcurrentPing)
		var wg sync.WaitGroup
		for rows.Next() {
			var id, port int
			var address string
			rows.Scan(&id, &address, &port)
			wg.Add(1)
			go func(nid int, addr string, p int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				start := time.Now()
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr, p), 2*time.Second)
				ping := -1
				if err == nil {
					ping = int(time.Since(start).Milliseconds())
					conn.Close()
				}
				if _, err := db.Exec("UPDATE nodes SET ping=? WHERE id=?", ping, nid); err != nil {
					log.Printf("[WARN] update node ping failed id=%d err=%v", nid, err)
				}
			}(id, address, port)
		}
		go func() {
			wg.Wait()
		}()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.PUT("/nodes/:id", func(c *gin.Context) {
		var n struct {
			Name, Group, Type, Address, UUID, Params string
			Port                                     int
		}
		if c.BindJSON(&n) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		if n.Port <= 0 {
			n.Port = 443
		}
		if n.Params == "" {
			n.Params = "{}"
		}
		if _, err := db.Exec("UPDATE nodes SET name=?, grp=?, type=?, address=?, port=?, uuid=?, params=? WHERE id=?", n.Name, n.Group, n.Type, n.Address, n.Port, n.UUID, n.Params, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.DELETE("/nodes/:id", func(c *gin.Context) {
		if _, err := db.Exec("DELETE FROM nodes WHERE id=?", c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.PUT("/nodes/:id/toggle", func(c *gin.Context) {
		if _, err := db.Exec("UPDATE nodes SET active = NOT active WHERE id=?", c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}