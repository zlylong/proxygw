with open('main.go', 'r') as f:
    code = f.read()

old_login = '''	api.POST("/login", func(c *gin.Context) {
		var pwd string
		db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&pwd)
		var req struct{ Username, Password string }
		if c.BindJSON(&req) == nil && req.Username == "admin" && req.Password == pwd {
			c.JSON(200, gin.H{"token": sessionToken})
		} else {
			c.JSON(401, gin.H{"error": "Unauthorized"})
		}
	})'''

new_login = '''	api.POST("/login", func(c *gin.Context) {
		var req struct{ Password string }
		if c.BindJSON(&req) == nil {
			var pwd string
			db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&pwd)
			if req.Password == pwd {
				c.JSON(200, gin.H{"token": sessionToken})
			} else {
				c.Status(401)
			}
		} else {
			c.Status(400)
		}
	})'''

code = code.replace(old_login, new_login)

with open('main.go', 'w') as f:
    f.write(code)
