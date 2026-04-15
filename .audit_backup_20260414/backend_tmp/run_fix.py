
import re

with open('main.go', 'r') as f:
    code = f.read()

# 1. formatUpstreams & Mosdns Config
format_upstreams = '''func formatUpstreams(addrs string, useSocks bool) string {
	parts := strings.Split(addrs, ",")
	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			if useSocks {
				res = append(res, fmt.Sprintf(`{ addr: "%s", socks5: "127.0.0.1:10808" }`, p))
			} else {
				res = append(res, fmt.Sprintf(`{ addr: "%s" }`, p))
			}
		}
	}
	if len(res) == 0 { return "[{ addr: \"114.114.114.114\" }]" }
	return "[" + strings.Join(res, ", ") + "]"
}

func applyMosdnsConfig() {'''
code = code.replace('func applyMosdnsConfig() {', format_upstreams)

mosdns_body_old = '''	var local, remote, lazyStr string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)'''
mosdns_body_new = '''	var local, remote, lazyStr, mode string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode)
	if mode == "" { mode = "smart" }'''
code = code.replace(mosdns_body_old, mosdns_body_new)

seq_old = '''      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_remote
`'''
seq_new = '''`
	if mode == "strict" {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_remote
`
	} else if mode == "fast" {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_local
`
	} else {
		seqStr += `      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $fallback
`
	}

	smartPlugins := ""
	if mode == "smart" {
		smartPlugins = `  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "/root/proxygw/core/mosdns/geoip_cn.txt"
  - tag: local_sequence
    type: sequence
    args:
      - exec: $forward_local
      - matches: [ "!resp_ip $geoip_cn" ]
        exec: drop_resp
  - tag: fallback
    type: fallback
    args:
      primary: local_sequence
      secondary: forward_remote
      threshold: 500
      always_standby: true
`
	}'''
code = code.replace(seq_old, seq_new)

mosdns_args_old = '''        - "geosite_cn.txt"
%s  - tag: forward_local
    type: forward
    args: { upstreams: [{ addr: "%s" }] }
  - tag: forward_remote
    type: forward
    args: { upstreams: [{ addr: "%s" }] }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, local, remote, seqStr)'''
mosdns_args_new = '''        - "/root/proxygw/core/mosdns/geosite_cn.txt"
%s%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''
code = code.replace(mosdns_args_old, mosdns_args_new)
code = code.replace('        - "proxy_domains.txt"', '        - "/root/proxygw/core/mosdns/proxy_domains.txt"')

with open('main.go', 'w') as f:
    f.write(code)
