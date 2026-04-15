import re
with open('main.go', 'r') as f:
    s = f.read()

s = s.replace('seqStr += "      - exec: \n"', 'seqStr += "      - exec: \\n"')
s = s.replace('seqStr += "      - exec: \n\t"', 'seqStr += "      - exec: \\n"')
s = re.sub(r'seqStr \+= "      - exec: $lazy_cache\r?\n"', 'seqStr += "      - exec: \\n"', s)
s = re.sub(r'seqStr \+= "      - exec: $lazy_cache\r?\n\t"', 'seqStr += "      - exec: \\n"', s)

with open('main.go', 'w') as f:
    f.write(s)
