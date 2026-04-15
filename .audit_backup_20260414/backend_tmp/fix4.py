with open('main.go', 'r') as f:
    text = f.read()

target = 'c.JSON(200, gin.H{token: proxyg...cret})'
replacement = 'c.JSON(200, gin.H{token: proxygw-token-secret})'

text = text.replace(target, replacement)

with open('main.go', 'w') as f:
    f.write(text)
