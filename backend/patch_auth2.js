const fs = require('fs');
let content = fs.readFileSync('/root/proxygw/backend/auth_routes.go', 'utf8');

const loginStart = `	public.POST("/login", func(c *gin.Context) {`;
const loginReplace = `	public.POST("/login", func(c *gin.Context) {
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

		var req struct{ Password string }`;

content = content.replace(loginStart + `\n\t\tvar req struct{ Password string }`, loginReplace);

const loginFail = `		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}`;
const loginFailReplace = `		if !ok {
			loginAttempts.Store(ip, attempts+1)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		loginAttempts.Delete(ip)`;

content = content.replace(loginFail, loginFailReplace);

fs.writeFileSync('/root/proxygw/backend/auth_routes.go', content);
