package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SessionInfo struct {
	ExpiresAt time.Time
}

var sessions sync.Map

func createSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	sessions.Store(token, SessionInfo{
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24-hour expiration
	})
	return token
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
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		token := createSession()
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

	authed.POST("/logout_all", func(c *gin.Context) {
		revokeAllSessions()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
