with open('main.go', 'r') as f:
    lines = f.readlines()
for i, line in enumerate(lines):
    if 'lines := strings.Split(string(xrayVersionOut),' in line:
        lines[i] = '\t\t\t\tlines := strings.Split(string(xrayVersionOut), "\\n")\n'
        if lines[i+1] == '"\n': lines[i+1] = ''
    if 'lines := strings.Split(strings.TrimSpace(string(out)),' in line:
        lines[i] = '\t\tlines := strings.Split(strings.TrimSpace(string(out)), "\\n")\n'
        if lines[i+1] == '"\n': lines[i+1] = ''

with open('main.go', 'w') as f:
    f.writelines(lines)
