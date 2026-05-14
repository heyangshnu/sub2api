package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthDeps optional probes for /health and /health/ready.
type HealthDeps struct {
	StoreType string
	Redis     *redis.Client
	SQLite    *sql.DB
}

// DetailedHealth returns process + dependency status (JSON).
func DetailedHealth(c *gin.Context, d HealthDeps) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	out := gin.H{
		"status":     "healthy",
		"store_type": d.StoreType,
		"checks":     gin.H{},
	}

	checks := gin.H{}
	allOK := true

	if d.Redis != nil {
		t0 := time.Now()
		err := d.Redis.Ping(ctx).Err()
		ms := time.Since(t0).Milliseconds()
		if err != nil {
			allOK = false
			checks["redis"] = gin.H{"ok": false, "error": err.Error(), "latency_ms": ms}
		} else {
			checks["redis"] = gin.H{"ok": true, "latency_ms": ms}
		}
	} else {
		checks["redis"] = gin.H{"ok": true, "skipped": true, "reason": "memory store"}
	}

	if d.SQLite != nil {
		t0 := time.Now()
		err := d.SQLite.PingContext(ctx)
		ms := time.Since(t0).Milliseconds()
		if err != nil {
			allOK = false
			checks["sqlite"] = gin.H{"ok": false, "error": err.Error(), "latency_ms": ms}
		} else {
			checks["sqlite"] = gin.H{"ok": true, "latency_ms": ms}
		}
	} else {
		checks["sqlite"] = gin.H{"ok": true, "skipped": true, "reason": "not configured"}
	}

	out["checks"] = checks
	if !allOK {
		out["status"] = "degraded"
	}
	c.JSON(http.StatusOK, out)
}

// ReadyHealth returns 503 if Redis is required but unreachable (K8s-style readiness).
func ReadyHealth(c *gin.Context, d HealthDeps) {
	if d.Redis == nil {
		c.JSON(http.StatusOK, gin.H{"ready": true, "store_type": d.StoreType})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := d.Redis.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready": false,
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true, "store_type": d.StoreType})
}
