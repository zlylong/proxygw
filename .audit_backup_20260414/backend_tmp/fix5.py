with open('main.go', 'r') as f:
    text = f.read()

# Try index replacement
idx = text.find('c.JSON(200, gin.H{"token": "proxyg')
if idx != -1:
    end_idx = text.find('})', idx) + 2
    text = text[:idx] + 'c.JSON(200, gin.H{"token": "proxygw-token-secret"})' + text[end_idx:]

with open('main.go', 'w') as f:
    f.write(text)
