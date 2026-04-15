import re
with open('main.go', 'r') as f:
    text = f.read()

text = text.replace('"proxyg...cret"', '"proxygw-token-secret"')

with open('main.go', 'w') as f:
    f.write(text)
