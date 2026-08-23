package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "request_id", util.RequestID(c), "error", fmt.Sprint(recovered), "stack", string(debug.Stack()))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "the request could not be completed"}})
			}
		}()
		c.Next()
	}
}
