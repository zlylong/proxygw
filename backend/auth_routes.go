package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"log"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SessionInfo struct {
	ExpiresAt time.Time
}

var sessions sync.Map
type LoginAttempt struct {
	Count    int
	LastSeen time.Time
}
var loginAttempts sync.Map

func createSession() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err // fail-close on entropy failure
	}
	token := hex.EncodeToString(b)
	sessions.Store(token, SessionInfo{
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24-hour expiration
	})
	return token, nil
}

func validateSession(token string) bool {
	val, ok := sessions.Load(token)
	if !ok {
		return false
	}
	info := val.(SessionInfo)
	if time.Now().After(info.ExpiresAt) {
		sessions.Delete(token)
		return false
	}
	// Slide expiration window
	sessions.Store(token, SessionInfo{
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	return true
}

func revokeAllSessions() {
	sessions.Range(func(key, value interface{}) bool {
		sessions.Delete(key)
		return true
	})
}

func clearSession(token string) {
	sessions.Delete(token)
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyAndMaybeMigratePassword(input string) (bool, error) {
	var hash string
	err := db.QueryRow("SELECT value FROM settings WHERE key='password_hash'").Scan(&hash)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if strings.TrimSpace(hash) != "" {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input)) == nil {
			return true, nil
		}
		return false, nil
	}

	var legacyPwd string
	err = db.QueryRow("SELECT value FROM settings WHERE key='password'").Scan(&legacyPwd)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if legacyPwd == "" || input != legacyPwd {
		return false, nil
	}

	newHash, err := hashPassword(input)
	if err != nil {
		return false, err
	}
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	if _, err = tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)", newHash); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if _, err = tx.Exec("DELETE FROM settings WHERE key='password'"); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func registerAuthRoutes(public *gin.RouterGroup, authed *gin.RouterGroup) {
	public.POST("/login", func(c *gin.Context) {
		ip := c.ClientIP()
		// clean up old entries periodically or inline
		now := time.Now()
		val, _ := loginAttempts.LoadOrStore(ip, LoginAttempt{Count: 0, LastSeen: now})
		attemptData := val.(LoginAttempt)
		if now.Sub(attemptData.LastSeen) > 30*time.Minute {
			attemptData.Count = 0
		}
		
		attempts := attemptData.Count
		if attempts > 10 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
			return
		}
		if attempts > 5 {
			time.Sleep(2 * time.Second)
		}
		attemptData.Count = attempts + 1
		attemptData.LastSeen = now
		loginAttempts.Store(ip, attemptData)

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
			// already incremented before verify
			log.Printf("Login failed for IP %s: incorrect password", ip)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
			return
		}
		loginAttempts.Delete(ip)
		token, err := createSession()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to generate session token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	authed.POST("/password", func(c *gin.Context) {
		var req struct{ Old, New string }
		if c.BindJSON(&req) != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		if len(strings.TrimSpace(req.New)) < 8 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "new password too short (min 8)"})
			return
		}

		ok, err := verifyAndMaybeMigratePassword(req.Old)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "old password mismatch"})
			return
		}

		hash, err := hashPassword(req.New)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "hash error"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err = tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)", hash); err != nil {
			_ = tx.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if _, err = tx.Exec("DELETE FROM settings WHERE key='password'"); err != nil {
			_ = tx.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if err = tx.Commit(); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		revokeAllSessions() // Force logout of all sessions
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	authed.POST("/logout", func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		clearSession(token)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

}
