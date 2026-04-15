package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerRuleRoutes(api *gin.RouterGroup) {
	api.GET("/rules/categories", func(c *gin.Context) {
		cacheMutex.Lock()
		if len(cachedGeosite) == 0 {
			cachedGeosite = parseDatFile("/usr/local/bin/geosite.dat")
		}
		if len(cachedGeoip) == 0 {
			cachedGeoip = parseDatFile("/usr/local/bin/geoip.dat")
		}
		resGeosite := cachedGeosite
		resGeoip := cachedGeoip
		cacheMutex.Unlock()
		c.JSON(http.StatusOK, gin.H{"geosite": resGeosite, "geoip": resGeoip})
	})

	api.GET("/rules", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, type, value, policy FROM rules")
		defer rows.Close()
		var rules []map[string]interface{}
		for rows.Next() {
			var id int
			var rtype, value, policy string
			rows.Scan(&id, &rtype, &value, &policy)
			rules = append(rules, map[string]interface{}{"id": id, "type": rtype, "value": value, "policy": policy})
		}
		if rules == nil {
			rules = make([]map[string]interface{}, 0)
		}
		c.JSON(http.StatusOK, rules)
	})

	api.POST("/rules", func(c *gin.Context) {
		var r struct{ Type, Value, Policy string }
		if c.BindJSON(&r) == nil {
			db.Exec("INSERT INTO rules (type, value, policy) VALUES (?, ?, ?)", r.Type, r.Value, r.Policy)
			scheduleApply()
			c.JSON(http.StatusOK, gin.H{"success": true})
		}
	})

	api.DELETE("/rules/:id", func(c *gin.Context) {
		db.Exec("DELETE FROM rules WHERE id=?", c.Param("id"))
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
