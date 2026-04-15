const fs = require('fs');
let content = fs.readFileSync('/root/proxygw/backend/auth_routes.go', 'utf8');

const regex = /public\.POST\("\/login", func\(c \*gin\.Context\) \{([\s\S]*?)token := createSession\(\)/g;

content = content.replace(regex, `public.POST("/login", func(c *gin.Context) {
		ip := c.ClientIP()
		val, _ := loginAttempts.LoadOrStore(ip, 0)
		attempts := val.(int)
		if attempts > 10 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
			return
		}
		if attempts > 5 {
			time.Sleep(2 * time.Second)
		}

		var req struct{ Password string }
		if c.BindJSON(&req) != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		if strings.TrimSpace(req.Password) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "password required"})
			return
		}

		ok, err := verifyAndMaybeMigratePassword(req.Password)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !ok {
			loginAttempts.Store(ip, attempts+1)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		loginAttempts.Delete(ip)
		token := createSession()`);

fs.writeFileSync('/root/proxygw/backend/auth_routes.go', content);
