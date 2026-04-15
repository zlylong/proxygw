with open('main.go', 'r') as f:
    s = f.read()

s = s.replace('seqStr += "      - exec: \n"', 'seqStr += "      - exec: \\n"')
s = s.replace('lines := strings.Split(string(xrayVersionOut), "\n")', 'lines := strings.Split(string(xrayVersionOut), "\\n")')
s = s.replace('lines := strings.Split(strings.TrimSpace(string(out)), "\n")', 'lines := strings.Split(strings.TrimSpace(string(out)), "\\n")')

with open('main.go', 'w') as f:
    f.write(s)
