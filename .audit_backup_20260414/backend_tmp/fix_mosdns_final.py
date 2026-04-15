import re
with open('main.go', 'r') as f:
    code = f.read()

# find the whole applyMosdnsConfig() { ... } function
pattern = r'func applyMosdnsConfig\(\) \{.*?\nos\.WriteFile\("\.\./core/mosdns/config\.yaml", \[\]byte\(config\), 0644\)\n\texec\.Command\("systemctl", "restart", "mosdns"\)\.Run\(\)\n\}'

new_func = '''func applyMosdnsConfig() {
	var local, remote, lazyStr, mode string
	db.QueryRow("SELECT value FROM settings WHERE key='dns_local'").Scan(&local)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_remote'").Scan(&remote)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_lazy'").Scan(&lazyStr)
	db.QueryRow("SELECT value FROM settings WHERE key='dns_mode'").Scan(&mode)
	if mode == "" { mode = "smart" }

	dRows, _ := db.Query("SELECT value FROM rules WHERE type='domain' AND policy LIKE 'proxy%'")
	var proxyDomains []string
	for dRows.Next() {
		var d string
		dRows.Scan(&d)
		proxyDomains = append(proxyDomains, d)
	}
	dRows.Close()
	os.WriteFile("../core/mosdns/proxy_domains.txt", []byte(strings.Join(proxyDomains, "\n")), 0644)

	lazyCache := ""
	if lazyStr == "true" {
		lazyCache = 
	}

	seqStr := 
	if lazyStr == "true" {
		seqStr += "      - exec: \n"
	}
	if mode == "strict" {
		seqStr += 
	} else if mode == "fast" {
		seqStr += 
	} else { // smart fallback
		seqStr += 
	}

	smartPlugins := ""
	if mode == "smart" {
		smartPlugins = 
	}

	config := fmt.Sprintf(, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)

	os.WriteFile("../core/mosdns/config.yaml", []byte(config), 0644)
	exec.Command("systemctl", "restart", "mosdns").Run()
}'''

code = re.sub(pattern, new_func, code, flags=re.DOTALL)
with open('main.go', 'w') as f:
    f.write(code)

