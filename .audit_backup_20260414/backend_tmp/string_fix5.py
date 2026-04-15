import re
with open('main.go', 'r') as f:
    s = f.read()

s = re.sub(r'seqStr \+= "      - exec: $lazy_cache\n\s*"', 'seqStr += "      - exec: \\n"', s)
with open('main.go', 'w') as f:
    f.write(s)
