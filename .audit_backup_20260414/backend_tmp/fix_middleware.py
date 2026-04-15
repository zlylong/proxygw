with open('main.go', 'r') as f:
    code = f.read()

old_mw = '''	api.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer "+sessionToken {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	})'''

new_mw = '''	api.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/api/login" {
			return
		}
		if c.GetHeader("Authorization") != "Bearer "+sessionToken {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	})'''

code = code.replace(old_mw, new_mw)

with open('main.go', 'w') as f:
    f.write(code)
