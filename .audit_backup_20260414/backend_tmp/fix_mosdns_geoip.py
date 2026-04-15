with open('main.go', 'r') as f:
    code = f.read()

# 1. Update updateGeoData()
old_dl1 = '''	err1 := downloadAndVerify(baseUrl+"direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
	err2 := downloadAndVerify(baseUrl+"geoip.dat", "/tmp/geoip.dat", 1024*1024)
	err3 := downloadAndVerify(baseUrl+"geosite.dat", "/tmp/geosite.dat", 1024*1024)

	if err1 == nil && err2 == nil && err3 == nil {
		exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
		exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
		exec.Command("cp", "/tmp/geosite.dat", "/usr/local/bin/geosite.dat").Run()
		exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
		exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()'''

new_dl1 = '''	err1 := downloadAndVerify(baseUrl+"direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
	err2 := downloadAndVerify(baseUrl+"geoip.dat", "/tmp/geoip.dat", 1024*1024)
	err3 := downloadAndVerify(baseUrl+"geosite.dat", "/tmp/geosite.dat", 1024*1024)
	err4 := downloadAndVerify("https://raw.githubusercontent.com/Loyalsoldier/geoip/release/text/cn.txt", "/tmp/geoip_cn.txt", 10*1024)

	if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
		exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
		exec.Command("cp", "/tmp/geosite.dat", "/usr/local/bin/geosite.dat").Run()
		exec.Command("cp", "/tmp/geoip_cn.txt", "/usr/local/bin/geoip_cn.txt").Run()
		exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
		exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()
		exec.Command("cp", "/usr/local/bin/geoip_cn.txt", "/root/proxygw/core/mosdns/").Run()'''
code = code.replace(old_dl1, new_dl1)

# 2. Update cronUpdater()
old_dl2 = '''			err1 := downloadAndVerify("https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
			err2 := downloadAndVerify("https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat", "/tmp/geoip.dat", 1024*1024)
			if err1 == nil && err2 == nil {
				exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
				exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
				exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
				exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()'''

new_dl2 = '''			err1 := downloadAndVerify("https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt", "/tmp/geosite_cn.txt", 10*1024)
			err2 := downloadAndVerify("https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat", "/tmp/geoip.dat", 1024*1024)
			err4 := downloadAndVerify("https://raw.githubusercontent.com/Loyalsoldier/geoip/release/text/cn.txt", "/tmp/geoip_cn.txt", 10*1024)
			if err1 == nil && err2 == nil && err4 == nil {
				exec.Command("cp", "/tmp/geosite_cn.txt", "/usr/local/bin/geosite_cn.txt").Run()
				exec.Command("cp", "/tmp/geoip.dat", "/usr/local/bin/geoip.dat").Run()
				exec.Command("cp", "/tmp/geoip_cn.txt", "/usr/local/bin/geoip_cn.txt").Run()
				exec.Command("cp", "/usr/local/bin/geosite_cn.txt", "/root/proxygw/core/mosdns/").Run()
				exec.Command("cp", "/usr/local/bin/geoip.dat", "/root/proxygw/core/mosdns/").Run()
				exec.Command("cp", "/usr/local/bin/geoip_cn.txt", "/root/proxygw/core/mosdns/").Run()'''
code = code.replace(old_dl2, new_dl2)

# 3. Update applyMosdnsConfig()
old_yaml = '''  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "geoip.dat"'''

new_yaml = '''  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "/root/proxygw/core/mosdns/geoip_cn.txt"'''
code = code.replace(old_yaml, new_yaml)

with open('main.go', 'w') as f:
    f.write(code)

