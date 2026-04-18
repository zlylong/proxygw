package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Subscription struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	AutoUpdate     int    `json:"auto_update"`
	UpdateInterval int    `json:"update_interval"`
	LastUpdate     string `json:"last_update"`
	Active         int    `json:"active"`
}

func registerSubscriptionRoutes(r *gin.RouterGroup) {
	api := r.Group("/subscriptions")

	api.GET("", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, url, auto_update, update_interval, IFNULL(last_update, ''), active FROM subscriptions")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var subs []Subscription
		for rows.Next() {
			var s Subscription
			if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.AutoUpdate, &s.UpdateInterval, &s.LastUpdate, &s.Active); err == nil {
				subs = append(subs, s)
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": subs})
	})

	api.POST("", func(c *gin.Context) {
		var s Subscription
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if s.UpdateInterval == 0 {
			s.UpdateInterval = 1440
		}
		
		var id int
		err := db.QueryRow("INSERT INTO subscriptions (name, url, auto_update, update_interval, active) VALUES (?, ?, ?, ?, ?) RETURNING id",
			s.Name, s.URL, s.AutoUpdate, s.UpdateInterval, s.Active).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.ID = id
		c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
	})

	api.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		db.Exec("DELETE FROM subscriptions WHERE id = ?", id)
		db.Exec("DELETE FROM nodes WHERE subscription_id = ?", id)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	api.POST("/:id/sync", func(c *gin.Context) {
		id := c.Param("id")
		var url string
		err := db.QueryRow("SELECT url FROM subscriptions WHERE id = ?", id).Scan(&url)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}

		err = syncSubscription(id, url)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		now := time.Now().Format("2006-01-02 15:04:05")
		db.Exec("UPDATE subscriptions SET last_update = ? WHERE id = ?", now, id)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
