with open('main.go', 'r') as f:
    s = f.read()

s = s.replace('ips := strings.Split(strings.TrimSpace(string(ipOut)), "\n")', 'ips := strings.Split(strings.TrimSpace(string(ipOut)), "\\n")')
s = s.replace('os.WriteFile("../core/mosdns/proxy_domains.txt", []byte(strings.Join(proxyDomains, "\n")), 0644)', 'os.WriteFile("../core/mosdns/proxy_domains.txt", []byte(strings.Join(proxyDomains, "\\n")), 0644)')
s = s.replace('seqStr += "      - exec: \n"', 'seqStr += "      - exec: \\n"')

with open('main.go', 'w') as f:
    f.write(s)
