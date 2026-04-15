with open('main.go', 'r') as f:
    content = f.read()

# 1. inbounds
old_inbounds = '''	"inbounds": []map[string]interface{}{
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
		},'''

new_inbounds = '''	"inbounds": []map[string]interface{}{
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
			{
				"listen": "127.0.0.1", "port": 10808, "protocol": "socks",
				"settings": map[string]interface{}{"auth": "noauth", "udp": true},
				"tag":      "dns_socks_inbound",
			},
		},'''

content = content.replace(old_inbounds, new_inbounds)

# 2. rules
old_rules = '''"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
			},'''

new_rules = '''"rules": []map[string]interface{}{
				{"inboundTag": []string{"api_inbound"}, "outboundTag": "api", "type": "field"},
				{"inboundTag": []string{"dns_socks_inbound"}, "outboundTag": "proxy", "type": "field"},
			},'''

content = content.replace(old_rules, new_rules)

# 3. mosdns formatUpstreams
old_format = '''func formatUpstreams(addrs string) string {
	parts := strings.Split(addrs, ",")
	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, fmt.Sprintf("{ addr: \"%s\" }", p))
		}
	}
	if len(res) == 0 {
		return "[{ addr: \"114.114.114.114\" }]" // fallback
	}
	return "[" + strings.Join(res, ", ") + "]"
}'''

new_format = '''func formatUpstreams(addrs string, useSocks bool) string {
	parts := strings.Split(addrs, ",")
	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			if useSocks {
				res = append(res, fmt.Sprintf("{ addr: \"%s\", socks5: \"127.0.0.1:10808\" }", p))
			} else {
				res = append(res, fmt.Sprintf("{ addr: \"%s\" }", p))
			}
		}
	}
	if len(res) == 0 {
		return "[{ addr: \"114.114.114.114\" }]" // fallback
	}
	return "[" + strings.Join(res, ", ") + "]"
}'''

content = content.replace(old_format, new_format)

# 4. apply calls
content = content.replace('formatUpstreams(local)', 'formatUpstreams(local, false)')
content = content.replace('formatUpstreams(remote)', 'formatUpstreams(remote, true)')

with open('main.go', 'w') as f:
    f.write(content)
