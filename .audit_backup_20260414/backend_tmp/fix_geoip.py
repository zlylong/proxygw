with open('main.go', 'r') as f:
    s = f.read()

s = s.replace('''  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "/root/proxygw/core/mosdns/geoip.dat"''', '''  - tag: geoip_cn
    type: ip_set
    args:
      files:
        - "/root/proxygw/core/mosdns/geoip_cn.txt"''')

with open('main.go', 'w') as f:
    f.write(s)
