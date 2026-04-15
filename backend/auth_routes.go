package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerAuthRoutes(api *gin.RouterGroup) {
	api.POST("/login", func(c *gin.Context) {
		var req struct{ Password string }
		if c.BindJSON(&req) != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		var pwd string
		err := db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&pwd)
		if err != nil && err != sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if req.Password != pwd {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": sessionToken})
	})

	api.POST("/password", func(c *gin.Context) {
		var req struct{ Old, New string }
		if c.BindJSON(&req) != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		var pwd string
		err := db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&pwd)
		if err != nil && err != sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if req.Old != pwd {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "old password mismatch"})
			return
		}
		if req.New == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "new password empty"})
			return
		}
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('password', ?)", req.New)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

}
