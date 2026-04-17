package main

import (
	"net/url"
	"encoding/json"
	"fmt"
	"net/http"
	"proxygw/remote_deploy"
	"github.com/gin-gonic/gin"
)

type RemoteNodeReq struct {
	Name          string `json:"name"`
	Type          string `json:"type"` // "wg" or "vless"
	SSHHost       string `json:"ssh_host"`
	SSHPort       int    `json:"ssh_port"`
	SSHUser       string `json:"ssh_user"`
	SSHAuthType   string `json:"ssh_auth_type"`
	SSHCredential string `json:"ssh_credential"`
	Region        string `json:"region"`
	Remark        string `json:"remark"`
}

func registerRemoteNodeRoutes(authed *gin.RouterGroup) {
	db.Exec("CREATE TABLE IF NOT EXISTS remote_node_history (id INTEGER PRIMARY KEY AUTOINCREMENT, node_id INTEGER, type TEXT, params TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(node_id) REFERENCES remote_nodes(id) ON DELETE CASCADE);")

	authed.GET("/remote_nodes", getRemoteNodes)
	authed.GET("/remote_nodes/:id", getRemoteNodeDetails)
	authed.POST("/remote_nodes", createAndDeployRemoteNode)
	authed.DELETE("/remote_nodes/:id", deleteRemoteNode)
	authed.POST("/remote_nodes/:id/check", checkRemoteNode)
	
	// Advanced Features
	authed.POST("/remote_nodes/batch", batchDeployRemoteNodes)
	authed.POST("/remote_nodes/:id/regenerate", regenerateRemoteNodeParams)
	authed.GET("/remote_nodes/:id/history", getRemoteNodeHistory)
	authed.POST("/remote_nodes/:id/rollback", rollbackRemoteNode)
}

func getRemoteNodes(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, type, ssh_host, region, status, remark, created_at FROM remote_nodes ORDER BY id DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var id int
		var name, ntype, host, region, status, remark, createdAt string
		if err := rows.Scan(&id, &name, &ntype, &host, &region, &status, &remark, &createdAt); err != nil {
			continue
		}
		nodes = append(nodes, map[string]interface{}{
			"id": id, "name": name, "type": ntype, "ssh_host": host,
			"region": region, "status": status, "remark": remark, "created_at": createdAt,
		})
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, nodes)
}

func getRemoteNodeDetails(c *gin.Context) {
	id := c.Param("id")
	var node map[string]interface{} = make(map[string]interface{})

	var ntype, name, host, region, status, remark string
	var port int
	err := db.QueryRow("SELECT name, type, ssh_host, ssh_port, region, status, remark FROM remote_nodes WHERE id = ?", id).
		Scan(&name, &ntype, &host, &port, &region, &status, &remark)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	
	node["id"] = id
	node["name"] = name
	node["type"] = ntype
	node["ssh_host"] = host
	node["ssh_port"] = port
	node["region"] = region
	node["status"] = status
	node["remark"] = remark

	if ntype == "wg" {
		var spriv, spub, cpriv, cpub, ep, taddr, caddr string
		var lport int
		err = db.QueryRow("SELECT server_priv, server_pub, client_priv, client_pub, endpoint, port, tunnel_addr, client_addr FROM remote_node_wg WHERE node_id = ?", id).
			Scan(&spriv, &spub, &cpriv, &cpub, &ep, &lport, &taddr, &caddr)
		if err == nil {
			node["wg"] = map[string]interface{}{
				"server_pub": spub, "client_priv": cpriv, "client_pub": cpub,
				"endpoint": ep, "port": lport, "tunnel_addr": taddr, "client_addr": caddr,
				"server_priv": spriv, "share_link": remote_deploy.GenerateWireGuardShareLink(cpriv, host, lport, spub, caddr, "", "ProxyGW-"+host, 1420), 
			}
		}
	} else if ntype == "vless" {
		var uuid, rpriv, rpub, sid, sname, dest, slink string
		var lport int
		err = db.QueryRow("SELECT uuid, reality_priv, reality_pub, short_id, server_name, dest, port, share_link FROM remote_node_vless WHERE node_id = ?", id).
			Scan(&uuid, &rpriv, &rpub, &sid, &sname, &dest, &lport, &slink)
		if err == nil {
			node["vless"] = map[string]interface{}{
				"uuid": uuid, "reality_pub": rpub, "short_id": sid,
				"server_name": sname, "dest": dest, "port": lport, "share_link": slink,
				"reality_priv": rpriv, 
			}
		}
	}

	c.JSON(http.StatusOK, node)
}

func logAction(nodeId int64, action, status, logText string) {
	db.Exec("INSERT INTO remote_node_logs (node_id, action, status, log_text) VALUES (?, ?, ?, ?)", nodeId, action, status, logText)
}

func doDeployRoutine(id int64, req RemoteNodeReq, isUpdate bool, params map[string]interface{}) {
	logAction(id, "deploy", "running", "Connecting via SSH...")
	
	sshClient, err := remote_deploy.Connect(req.SSHHost, req.SSHPort, req.SSHUser, req.SSHAuthType, req.SSHCredential)
	if err != nil {
		db.Exec("UPDATE remote_nodes SET status = 'Failed' WHERE id = ?", id)
		logAction(id, "deploy", "failed", err.Error())
		return
	}
	defer sshClient.Close()
	
	logAction(id, "deploy", "running", "Connected successfully, generating parameters...")
	
	var script string

	if req.Type == "wg" {
		var sPriv, sPub, cPriv, cPub, tunnel, clientIP string
		var port int
		
		if params == nil {
			port, _ = remote_deploy.GenerateUniquePort(db, 10000, 60000)
			sPriv, sPub, _ = remote_deploy.GenerateWireGuardKeys()
			cPriv, cPub, _ = remote_deploy.GenerateWireGuardKeys()
			tunnel, clientIP, _ = remote_deploy.GenerateUniqueWGTunnel(db)
		} else {
			port = int(params["port"].(float64))
			sPriv, sPub = params["server_priv"].(string), params["server_pub"].(string)
			cPriv, cPub = params["client_priv"].(string), params["client_pub"].(string)
			tunnel, clientIP = params["tunnel_addr"].(string), params["client_addr"].(string)
		}
		
		endpoint := fmt.Sprintf("%s:%d", req.SSHHost, port)
		
		if !isUpdate {
			db.Exec("INSERT INTO remote_node_wg (node_id, server_priv, server_pub, client_priv, client_pub, endpoint, port, tunnel_addr, client_addr) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				id, sPriv, sPub, cPriv, cPub, endpoint, port, tunnel, clientIP)
		} else {
			db.Exec("UPDATE remote_node_wg SET server_priv=?, server_pub=?, client_priv=?, client_pub=?, endpoint=?, port=?, tunnel_addr=?, client_addr=? WHERE node_id=?",
				sPriv, sPub, cPriv, cPub, endpoint, port, tunnel, clientIP, id)
		}
			
		script = remote_deploy.GenerateWGInstallScript(port, sPriv, cPub, tunnel)
		
	} else if req.Type == "vless" {
		var rPriv, rPub, uuid, shortId, dest, serverName, shareLink string
		var port int
		
		if params == nil {
			port, _ = remote_deploy.GenerateUniquePort(db, 10000, 60000)
			rPriv, rPub, _ = remote_deploy.GenerateXrayRealityKeys()
			uuid = remote_deploy.GenerateUUID()
			shortId, _ = remote_deploy.GenerateShortId()
			dest = "www.microsoft.com:443"
			serverName = "www.microsoft.com"
		} else {
			port = int(params["port"].(float64))
			rPriv, rPub = params["reality_priv"].(string), params["reality_pub"].(string)
			uuid, shortId = params["uuid"].(string), params["short_id"].(string)
			dest, serverName = params["dest"].(string), params["server_name"].(string)
		}
		
		shareLink = fmt.Sprintf("vless://%s@%s:%d?security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=tcp&flow=xtls-rprx-vision&encryption=none#%s",
			uuid, req.SSHHost, port, serverName, rPub, shortId, url.QueryEscape(req.Name))
			
		if !isUpdate {
			db.Exec("INSERT INTO remote_node_vless (node_id, uuid, reality_priv, reality_pub, short_id, server_name, dest, port, share_link) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				id, uuid, rPriv, rPub, shortId, serverName, dest, port, shareLink)
		} else {
			db.Exec("UPDATE remote_node_vless SET uuid=?, reality_priv=?, reality_pub=?, short_id=?, server_name=?, dest=?, port=?, share_link=? WHERE node_id=?",
				uuid, rPriv, rPub, shortId, serverName, dest, port, shareLink, id)
		}
			
		script = remote_deploy.GenerateVlessRealityInstallScript(port, uuid, rPriv, shortId, serverName, dest)
	}
	
	logAction(id, "deploy", "running", "Executing installation script on remote host...")
	stdout, stderr, err := sshClient.RunCommand(script)
	
	if err != nil {
		db.Exec("UPDATE remote_nodes SET status = 'Failed' WHERE id = ?", id)
		logAction(id, "deploy", "failed", fmt.Sprintf("Error: %v\nStderr: %s", err, stderr))
		return
	}
	
	db.Exec("UPDATE remote_nodes SET status = 'Online' WHERE id = ?", id)
	logAction(id, "deploy", "success", fmt.Sprintf("Deployment successful.\nStdout: %s", stdout))
}

func createAndDeployRemoteNode(c *gin.Context) {
	var req RemoteNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := db.Exec("INSERT INTO remote_nodes (name, type, ssh_host, ssh_port, ssh_user, ssh_auth_type, ssh_credential, region, status, remark) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'Deploying', ?)",
		req.Name, req.Type, req.SSHHost, req.SSHPort, req.SSHUser, req.SSHAuthType, EncryptAES(req.SSHCredential), req.Region, req.Remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert node"})
		return
	}
	nodeId, _ := res.LastInsertId()
	go doDeployRoutine(nodeId, req, false, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Deployment started", "id": nodeId})
}

func batchDeployRemoteNodes(c *gin.Context) {
	var reqs []RemoteNodeReq
	if err := c.ShouldBindJSON(&reqs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, req := range reqs {
		res, err := db.Exec("INSERT INTO remote_nodes (name, type, ssh_host, ssh_port, ssh_user, ssh_auth_type, ssh_credential, region, status, remark) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'Deploying', ?)",
			req.Name, req.Type, req.SSHHost, req.SSHPort, req.SSHUser, req.SSHAuthType, EncryptAES(req.SSHCredential), req.Region, req.Remark)
		if err == nil {
			nodeId, _ := res.LastInsertId()
			go doDeployRoutine(nodeId, req, false, nil)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("Batch deployment started for %d nodes", len(reqs))})
}

func deleteRemoteNode(c *gin.Context) {
	id := c.Param("id")
	
	req, err := fetchNodeReq(id)
	if err == nil {
		go func(req RemoteNodeReq) {
			client, err := remote_deploy.Connect(req.SSHHost, req.SSHPort, req.SSHUser, req.SSHAuthType, req.SSHCredential)
			if err == nil {
				defer client.Close()
				if req.Type == "wg" {
					client.RunCommand("systemctl stop wg-quick@wg0; systemctl disable wg-quick@wg0; rm -f /etc/wireguard/wg0.conf")
				} else if req.Type == "vless" {
					client.RunCommand("systemctl stop xray; systemctl disable xray; rm -f /etc/systemd/system/xray.service; rm -rf /usr/local/etc/xray; rm -f /usr/local/bin/xray")
				}
			}
		}(req)
	}

	db.Exec("DELETE FROM remote_node_wg WHERE node_id = ?", id)
	db.Exec("DELETE FROM remote_node_vless WHERE node_id = ?", id)
	db.Exec("DELETE FROM remote_node_logs WHERE node_id = ?", id)
	db.Exec("DELETE FROM remote_node_history WHERE node_id = ?", id)
	_, err = db.Exec("DELETE FROM remote_nodes WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func checkRemoteNode(c *gin.Context) {
	id := c.Param("id")
	var host, authType, credential, user, ntype string
	var port int
	err := db.QueryRow("SELECT ssh_host, ssh_port, ssh_user, ssh_auth_type, ssh_credential, type FROM remote_nodes WHERE id = ?", id).
		Scan(&host, &port, &user, &authType, &credential, &ntype)
	credential = DecryptAES(credential)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	
	client, err := remote_deploy.Connect(host, port, user, authType, credential)
	if err != nil {
		db.Exec("UPDATE remote_nodes SET status = 'Offline' WHERE id = ?", id)
		logAction(0, "check", "failed", fmt.Sprintf("Node %s SSH check failed: %v", id, err))
		c.JSON(http.StatusOK, gin.H{"success": false, "status": "Offline", "reason": err.Error()})
		return
	}
	defer client.Close()
	
	cmd := "systemctl is-active xray"
	if ntype == "wg" { cmd = "systemctl is-active wg-quick@wg0" }
	
	out, _, err := client.RunCommand(cmd)
	status := "Online"
	if err != nil || out == "" { status = "Offline" }
	
	db.Exec("UPDATE remote_nodes SET status = ? WHERE id = ?", status, id)
	c.JSON(http.StatusOK, gin.H{"success": true, "status": status})
}

func fetchNodeReq(id string) (RemoteNodeReq, error) {
	var req RemoteNodeReq
	err := db.QueryRow("SELECT name, type, ssh_host, ssh_port, ssh_user, ssh_auth_type, ssh_credential, region, remark FROM remote_nodes WHERE id = ?", id).
		Scan(&req.Name, &req.Type, &req.SSHHost, &req.SSHPort, &req.SSHUser, &req.SSHAuthType, &req.SSHCredential, &req.Region, &req.Remark)
	req.SSHCredential = DecryptAES(req.SSHCredential)
	return req, err
}

func regenerateRemoteNodeParams(c *gin.Context) {
	id := c.Param("id")
	req, err := fetchNodeReq(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	// Archive old params
	var oldParams map[string]interface{} = make(map[string]interface{})
	if req.Type == "wg" {
		var spriv, spub, cpriv, cpub, ep, taddr, caddr string
		var lport int
		if err := db.QueryRow("SELECT server_priv, server_pub, client_priv, client_pub, endpoint, port, tunnel_addr, client_addr FROM remote_node_wg WHERE node_id = ?", id).
			Scan(&spriv, &spub, &cpriv, &cpub, &ep, &lport, &taddr, &caddr); err == nil {
			oldParams = map[string]interface{}{"server_priv": spriv, "share_link": remote_deploy.GenerateWireGuardShareLink(cpriv, req.SSHHost, lport, spub, caddr, "", "ProxyGW-"+req.SSHHost, 1420), "server_pub": spub, "client_priv": cpriv, "client_pub": cpub, "endpoint": ep, "port": lport, "tunnel_addr": taddr, "client_addr": caddr}
		}
	} else {
		var uuid, rpriv, rpub, sid, sname, dest, slink string
		var lport int
		if err := db.QueryRow("SELECT uuid, reality_priv, reality_pub, short_id, server_name, dest, port, share_link FROM remote_node_vless WHERE node_id = ?", id).
			Scan(&uuid, &rpriv, &rpub, &sid, &sname, &dest, &lport, &slink); err == nil {
			oldParams = map[string]interface{}{"uuid": uuid, "reality_priv": rpriv, "reality_pub": rpub, "short_id": sid, "server_name": sname, "dest": dest, "port": lport, "share_link": slink}
		}
	}
	
	paramsJSON, _ := json.Marshal(oldParams)
	db.Exec("INSERT INTO remote_node_history (node_id, type, params) VALUES (?, ?, ?)", id, req.Type, string(paramsJSON))

	db.Exec("UPDATE remote_nodes SET status = 'Deploying' WHERE id = ?", id)
	
	var intId int64
	fmt.Sscanf(id, "%d", &intId)
	
	go doDeployRoutine(intId, req, true, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Regeneration started"})
}

func getRemoteNodeHistory(c *gin.Context) {
	id := c.Param("id")
	rows, err := db.Query("SELECT id, params, created_at FROM remote_node_history WHERE node_id = ? ORDER BY id DESC", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var hid int
		var pjson, cat string
		if err := rows.Scan(&hid, &pjson, &cat); err == nil {
			history = append(history, map[string]interface{}{"id": hid, "params": pjson, "created_at": cat})
		}
	}
	if history == nil { history = []map[string]interface{}{} }
	c.JSON(http.StatusOK, history)
}

func rollbackRemoteNode(c *gin.Context) {
	id := c.Param("id")
	var reqBody struct { HistoryId int `json:"history_id"` }
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := fetchNodeReq(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	var pjson string
	if err := db.QueryRow("SELECT params FROM remote_node_history WHERE id = ? AND node_id = ?", reqBody.HistoryId, id).Scan(&pjson); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History record not found"})
		return
	}

	var oldParams map[string]interface{}
	json.Unmarshal([]byte(pjson), &oldParams)

	db.Exec("UPDATE remote_nodes SET status = 'Deploying' WHERE id = ?", id)
	
	var intId int64
	fmt.Sscanf(id, "%d", &intId)
	
	go doDeployRoutine(intId, req, true, oldParams)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Rollback started"})
}
