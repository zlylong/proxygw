with open('main.go', 'r') as f:
    content = f.read()

# fix the broken string arrays
content = content.replace('tables := []string{\n\t\t,\n\t\t,\n\t\t,\n\t\t,\n\t}', '''tables := []string{
		,
		,
		,
		,
	}''')

with open('main.go', 'w') as f:
    f.write(content)
