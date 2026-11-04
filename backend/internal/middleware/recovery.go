package middleware

import (
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("请求发生 panic", zap.Any("panic", recovered), zap.String("request_id", GetRequestID(c)), zap.Stack("stack"))
				problem.Write(c, problem.Internal())
			}
		}()
		c.Next()
	}
}
