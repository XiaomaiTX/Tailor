package router

import (
	"net/http"

	"github.com/XiaomaiTX/go-react-starter/internal/api/generated"
	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/handler"
	"github.com/XiaomaiTX/go-react-starter/internal/middleware"
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New(api generated.StrictServerInterface, cfg config.Config, logger *zap.Logger, peerRelay ...*service.PeerRelayService) *gin.Engine {
	return NewWithAuth(api, cfg, logger, nil, peerRelay...)
}

func NewWithAuth(api generated.StrictServerInterface, cfg config.Config, logger *zap.Logger, oidcService *service.OIDCService, peerRelay ...*service.PeerRelayService) *gin.Engine {
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.CORSOrigins),
		middleware.Recovery(logger),
		middleware.AccessLogger(logger),
		middleware.AuthSession(oidcService, cfg.CORSOrigins, oidcService != nil && oidcService.SecureCookies()),
		middleware.APIRateLimit(),
	)
	if oidcService != nil {
		handler.RegisterAuthRoutes(engine, oidcService, logger)
	}
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
	for _, service := range peerRelay {
		if service != nil {
			handler.RegisterPeerRelayRoutes(engine, service)
		}
	}
	engine.NoRoute(func(c *gin.Context) { problem.Write(c, problem.NotFound()) })
	return engine
}
