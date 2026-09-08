package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func New(ping func(context.Context) error, frontendOrigin string) http.Handler {
	router := gin.New()
	router.Use(requestID(), requestLog(), gin.Recovery(), cors(frontendOrigin))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if err := ping(c.Request.Context()); err != nil {
			slog.Warn("database readiness check failed", "request_id", c.GetString("request_id"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"code": "not_ready", "message": "database unavailable",
			}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	return router
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		value := make([]byte, 16)
		if _, err := rand.Read(value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"code": "internal_error", "message": "request ID unavailable",
			}})
			return
		}
		id := hex.EncodeToString(value)
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		slog.Info("request",
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"duration", time.Since(started),
		)
	}
}

func cors(frontendOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == frontendOrigin {
			c.Header("Access-Control-Allow-Origin", frontendOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
