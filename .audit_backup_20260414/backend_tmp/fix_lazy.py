import os
with open('main.go', 'r') as f:
    lines = f.readlines()
for i, line in enumerate(lines):
    if 'seqStr +=' in line and '- exec: ' in line:
        lines[i] = '\t\tseqStr += "      - exec: \\n"\n'
        lines[i+1] = ''

with open('main.go', 'w') as f:
    f.writelines(lines)
