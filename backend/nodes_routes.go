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
		rows, err := db.Query("SELECT id, name, grp, type, address, port, uuid, active, ping, COALESCE(params, '{}') FROM nodes")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query error"})
			return
		}
		defer rows.Close()
		var nodes []map[string]interface{}
		for rows.Next() {
			var id, port, ping int
			var name, grp, ntype, address, uuid, params string
			var active bool
			if err := rows.Scan(&id, &name, &grp, &ntype, &address, &port, &uuid, &active, &ping, &params); err != nil {
				continue
			}
			nodes = append(nodes, map[string]interface{}{"id": id, "name": name, "group": grp, "type": ntype, "address": address, "port": port, "uuid": uuid, "active": active, "ping": ping, "params": params})
		}
		if nodes == nil {
			nodes = make([]map[string]interface{}, 0)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db rows error"})
			return
		}
		c.JSON(http.StatusOK, nodes)
	})

	api.POST("/nodes", func(c *gin.Context) {
		var n struct {
			Name, Group, Type, Address, UUID, Params string
			Port                             int
		}
		if c.BindJSON(&n) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		if n.Params == "" {
			n.Params = "{}"
		}
		if _, err := db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, ?, ?, ?, ?, ?, ?, 1)", n.Name, n.Group, n.Type, n.Address, n.Port, n.UUID, n.Params); err != nil {
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
				decoded, err := base64.StdEncoding.DecodeString(b64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vmess base64 content"})
					return
				}
				var v struct {
					Ps, Add, Id, Net, Tls, Sni, Path, Host string
					Port        interface{}
				}
				if err := json.Unmarshal(decoded, &v); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vmess payload"})
					return
				}
				if strings.TrimSpace(v.Add) == "" || strings.TrimSpace(v.Id) == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vmess endpoint"})
					return
				}
				portInt := parsePortValue(v.Port)
				if portInt <= 0 || portInt > 65535 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vmess port"})
					return
				}

				streamSettings := make(map[string]interface{})
				if v.Net != "" { streamSettings["network"] = v.Net }
				if v.Tls != "" { streamSettings["security"] = v.Tls }
				
				if v.Net == "ws" {
					wsSettings := make(map[string]interface{})
					if v.Path != "" { wsSettings["path"] = v.Path }
					if v.Host != "" { wsSettings["headers"] = map[string]string{"Host": v.Host} }
					if len(wsSettings) > 0 { streamSettings["wsSettings"] = wsSettings }
				} else if v.Net == "grpc" {
					if v.Path != "" { streamSettings["grpcSettings"] = map[string]string{"serviceName": v.Path} }
				}
				
				if v.Tls == "tls" {
					if v.Sni != "" { streamSettings["tlsSettings"] = map[string]string{"serverName": v.Sni} }
				}
				
				finalParamsVmess := map[string]interface{}{}
				if len(streamSettings) > 0 {
					finalParamsVmess["streamSettings"] = streamSettings
				}
				vmessParamsJson, err := json.Marshal(finalParamsVmess)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Marshal vmess params failed"})
					return
				}
				if _, err := db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, 'Imported', 'Vmess', ?, ?, ?, ?, 1)", v.Ps, v.Add, portInt, v.Id, string(vmessParamsJson)); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
					return
				}

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
					if strings.TrimSpace(host) == "" || strings.TrimSpace(uuid) == "" || portInt <= 0 || portInt > 65535 {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vless endpoint"})
						return
					}
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

					finalParams := map[string]interface{}{}
					if len(ss) > 0 {
						finalParams["streamSettings"] = ss
					}
					if f := params["flow"]; f != nil && f != "" {
						finalParams["flow"] = f
					}
					if e := params["encryption"]; e != nil && e != "" {
						finalParams["encryption"] = e
					}

					paramsJson, err := json.Marshal(finalParams)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Marshal vless params failed"})
						return
					}

					if _, err := db.Exec("INSERT INTO nodes (name, grp, type, address, port, uuid, params, active) VALUES (?, 'Imported', 'Vless', ?, ?, ?, ?, 1)", alias, host, portInt, uuid, string(paramsJson)); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
						return
					}
					scheduleApply()
					c.JSON(http.StatusOK, gin.H{"success": true})
					return
				}
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported URL format"})
	})

	api.POST("/nodes/ping", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, address, port FROM nodes")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query error"})
			return
		}
		defer rows.Close()
		const maxConcurrentPing = 20
		sem := make(chan struct{}, maxConcurrentPing)
		var wg sync.WaitGroup
		for rows.Next() {
			var id, port int
			var address string
			if err := rows.Scan(&id, &address, &port); err != nil {
				continue
			}
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
		if err := rows.Err(); err != nil {
			log.Printf("[ERROR] ping nodes rows err: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db rows error"})
			return
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
