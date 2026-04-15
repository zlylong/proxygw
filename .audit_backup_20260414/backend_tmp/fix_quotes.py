with open('main.go', 'r') as f:
    lines = f.readlines()

new_lines = []
for line in lines:
    if '{ addr: "%s", socks5: "127.0.0.1:10808" }' in line:
        pass
    if '{ addr: ' in line and 'socks5: ' in line:
        line = '\t\t\t\tres = append(res, fmt.Sprintf("{ addr: \"%s\", socks5: \"127.0.0.1:10808\" }", p))\n'
    elif '{ addr: ' in line and '114' not in line:
        line = '\t\t\t\tres = append(res, fmt.Sprintf("{ addr: \"%s\" }", p))\n'
    elif '114.114' in line:
        line = '\t\treturn "[{ addr: \"114.114.114.114\" }]" // fallback\n'
    new_lines.append(line)

with open('main.go', 'w') as f:
    f.writelines(new_lines)
