with open('main.go', 'r') as f:
    s = f.read()

s = s.replace('seqStr += "      - exec: \n"', 'seqStr += "      - exec: \\n"')
s = s.replace('seqStr += "      - exec: \n\t"', 'seqStr += "      - exec: \\n"')

with open('main.go', 'w') as f:
    f.write(s)
