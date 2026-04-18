package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func applySingboxConfig() error {
	rows, err := db.Query("SELECT id, name, address, port, uuid, COALESCE(params, '{}') FROM nodes WHERE active=1 AND (LOWER(type)='hysteria2' OR LOWER(type)='hy2')")
	if err != nil {
		return err
	}
	defer rows.Close()

	var inbounds []map[string]interface{}
	var outbounds []map[string]interface{}
	var rules []map[string]interface{}

	for rows.Next() {
		var id, port int
		var name, address, uuid, paramsStr string
		if err := rows.Scan(&id, &name, &address, &port, &uuid, &paramsStr); err != nil {
			continue
		}

		var params map[string]interface{}
		json.Unmarshal([]byte(paramsStr), &params)

		localPort := 20000 + id
		inboundTag := fmt.Sprintf("in-%d", id)
		outboundTag := fmt.Sprintf("out-%d", id)

		inbounds = append(inbounds, map[string]interface{}{
			"type":        "socks",
			"tag":         inboundTag,
			"listen":      "127.0.0.1",
			"listen_port": localPort,
		})

		outbound := map[string]interface{}{
			"type":     "hysteria2",
			"tag":      outboundTag,
			"server":   address,
			"server_port": port,
			"password": uuid,
		}

		tls := map[string]interface{}{}
		if sni, ok := params["sni"].(string); ok && sni != "" {
			tls["server_name"] = sni
		}
		if insecure, ok := params["insecure"].(string); ok && insecure == "1" {
			tls["insecure"] = true
		} else {
            tls["insecure"] = false
        }
		tls["enabled"] = true
		outbound["tls"] = tls

		if obfsPassword, ok := params["obfsPassword"].(string); ok && obfsPassword != "" {
			obfsType := "salamander"
			if ot, ok := params["obfs"].(string); ok && ot != "" {
				obfsType = ot
			}
			outbound["obfs"] = map[string]interface{}{
				"type":     obfsType,
				"password": obfsPassword,
			}
		}

		outbounds = append(outbounds, outbound)
		rules = append(rules, map[string]interface{}{
			"inbound":  inboundTag,
			"outbound": outboundTag,
		})
	}

	if len(inbounds) == 0 {
		exec.Command("systemctl", "stop", "singbox").Run()
		return nil
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level": "warn",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]interface{}{
			"rules": rules,
		},
	}

	b, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("/root/proxygw/core/singbox/config.json", b, 0644)

	return exec.Command("systemctl", "restart", "singbox").Run()
}
