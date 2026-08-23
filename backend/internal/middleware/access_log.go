package middleware

import (
	"log/slog"
	"time"

	"stage-rigging-cue-interlock/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func AccessLog(logger *slog.Logger, limiter *LocalRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		_, username, role := auth.Actor(c)
		logger.Info("http request",
			"request_id", "unknown",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"response_bytes", c.Writer.Size(),
			"actor", username,
			"role", role,
			"client_ip", c.ClientIP(),
			"rate_remaining", limiter.Remaining(c.ClientIP()),
		)
	}
}
