import re
with open('main.go', 'r') as f:
    s = f.read()

s = re.sub(r'exec: $lazy_cache\r?\n"', 'exec: \\n"', s)
s = re.sub(r'lines := strings\.Split\(string\(xrayVersionOut\), "\r?\n"\)', 'lines := strings.Split(string(xrayVersionOut), "\\n")', s)
s = re.sub(r'lines := strings\.Split\(strings\.TrimSpace\(string\(out\)\), "\r?\n"\)', 'lines := strings.Split(strings.TrimSpace(string(out)), "\\n")', s)

with open('main.go', 'w') as f:
    f.write(s)
