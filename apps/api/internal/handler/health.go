package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	status := http.StatusOK

	// DB ping
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		checks["database"] = "unhealthy"
		status = http.StatusServiceUnavailable
	} else {
		checks["database"] = "ok"
	}

	// Redis ping
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unhealthy"
		status = http.StatusServiceUnavailable
	} else {
		checks["redis"] = "ok"
	}

	c.JSON(status, gin.H{"status": checks})
}
