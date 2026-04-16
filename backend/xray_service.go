package main

func buildBaseXrayConfig() map[string]interface{} {
	return map[string]interface{}{
		"log":    map[string]string{"loglevel": "warning"},
		"api":    map[string]interface{}{"services": []string{"StatsService"}, "tag": "api"},
		"stats":  map[string]interface{}{},
		"policy": map[string]interface{}{"system": map[string]interface{}{"statsInboundDownlink": true, "statsInboundUplink": true}},
		"fakedns": []map[string]interface{}{
			{
				"id":       "fakedns",
				"ipPool":   "198.18.0.0/16",
				"poolSize": 65535,
			},
		},
		"dns": map[string]interface{}{
			"servers": []string{"fakedns"},
		},
		"inbounds": []map[string]interface{}{
			{
				"port": 12345, "protocol": "dokodemo-door",
				"settings":       map[string]interface{}{"network": "tcp,udp", "followRedirect": true},
				"streamSettings": map[string]interface{}{"sockopt": map[string]string{"tproxy": "tproxy"}},
				"sniffing":       map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls", "fakedns"}},
			},
			{
				"listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door",
				"settings": map[string]interface{}{"address": "127.0.0.1"},
				"tag":      "api_inbound",
			},
			{
				"port": 5353, "listen": "127.0.0.1", "protocol": "dokodemo-door",
				"settings": map[string]interface{}{"address": "8.8.8.8", "port": 53, "network": "udp"},
				"tag":      "dns-in",
			},
		},
		"outbounds": []map[string]interface{}{
			{"protocol": "freedom", "tag": "direct", "streamSettings": map[string]interface{}{"sockopt": map[string]interface{}{"mark": 2}}},
			{"protocol": "blackhole", "tag": "block"},
			{"protocol": "dns", "tag": "dns-out", "streamSettings": map[string]interface{}{"sockopt": map[string]interface{}{"mark": 2}}},
		},
		"routing": map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
				{"inboundTag": []string{"dns-in"}, "outboundTag": "dns-out", "type": "field"},
			},
		},
	}
}
