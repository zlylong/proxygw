import re
with open('main.go', 'r') as f:
    text = f.read()

# I will just write a completely new file for line 583
lines = text.split('\n')
for i, line in enumerate(lines):
    if 'c.JSON(200, gin.H{token: ' in line:
        lines[i] = '\t\t\t\tc.JSON(200, gin.H{token: proxygw-token-secret})'

with open('main.go', 'w') as f:
    f.write('\n'.join(lines))
