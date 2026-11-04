package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AccessLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		fields := []zap.Field{
			zap.String("request_id", GetRequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(started)),
		}
		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			logger.Error("请求完成", fields...)
		case c.Writer.Status() >= http.StatusBadRequest:
			logger.Warn("请求完成", fields...)
		default:
			logger.Info("请求完成", fields...)
		}
	}
}
