package main

func buildBaseXrayConfig() map[string]interface{} {
	return map[string]interface{}{
		"log":    map[string]string{"loglevel": "warning"},
		"api":    map[string]interface{}{"services": []string{"StatsService"}, "tag": "api"},
		"stats":  map[string]interface{}{},
		"policy": map[string]interface{}{"system": map[string]interface{}{"statsInboundDownlink": true, "statsInboundUplink": true}},
		"inbounds": []map[string]interface{}{
			{
				"port": 12345, "protocol": "dokodemo-door",
				"settings":       map[string]interface{}{"network": "tcp,udp", "followRedirect": true},
				"streamSettings": map[string]interface{}{"sockopt": map[string]string{"tproxy": "tproxy"}},
				"sniffing":       map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls"}},
			},
			{
				"listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door",
				"settings": map[string]interface{}{"address": "127.0.0.1"},
				"tag":      "api_inbound",
			},
		},
		"outbounds": []map[string]interface{}{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "block"},
		},
		"routing": map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
			},
		},
	}
}
