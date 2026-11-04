package router

import (
	"net/http"

	"github.com/XiaomaiTX/go-react-starter/internal/api/generated"
	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/middleware"
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New(api generated.StrictServerInterface, cfg config.Config, logger *zap.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.CORSOrigins),
		middleware.AccessLogger(logger),
		middleware.Recovery(logger),
	)
	strictHandler := generated.NewStrictHandler(api, nil)
	generated.RegisterHandlersWithOptions(engine, strictHandler, generated.GinServerOptions{
		BaseURL: "/api/v1",
		ErrorHandler: func(c *gin.Context, err error, status int) {
			if status == http.StatusBadRequest {
				problem.Write(c, problem.Validation("请求格式不正确"))
				return
			}
			logger.Error("OpenAPI 请求绑定失败", zap.Error(err))
			problem.Write(c, problem.Internal())
		},
	})
	engine.NoRoute(func(c *gin.Context) { problem.Write(c, problem.NotFound()) })
	return engine
}
