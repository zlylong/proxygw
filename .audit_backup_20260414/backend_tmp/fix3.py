with open('main.go', 'r') as f:
    text = f.read()

# Let's find out what the exact string is.
import json
print(repr(text[text.find('c.JSON(200'):text.find('c.JSON(200')+60]))
