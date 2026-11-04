package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/handler"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"github.com/XiaomaiTX/go-react-starter/internal/router"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)
	defer logger.Sync() //nolint:errcheck

	database, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接 PostgreSQL 失败", zap.Error(err))
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		logger.Fatal("获取 PostgreSQL 连接池失败", zap.Error(err))
	}
	defer sqlDatabase.Close() //nolint:errcheck

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("解析 Redis URL 失败", zap.Error(err))
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close() //nolint:errcheck

	exampleRepository := repository.NewExampleRepository(database)
	exampleService := service.NewExampleService(exampleRepository)
	api := handler.NewAPI(exampleService, sqlDatabase.PingContext, func(ctx context.Context) error {
		return redisClient.Ping(ctx).Err()
	})
	engine := router.New(api, cfg, logger)
	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.APIPort), Handler: engine, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		logger.Info("HTTP 服务已启动", zap.Int("port", cfg.APIPort))
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Fatal("HTTP 服务异常退出", zap.Error(serveErr))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("HTTP 服务关闭失败", zap.Error(err))
	}
}

func newLogger(level string) *zap.Logger {
	parsedLevel := zapcore.InfoLevel
	_ = parsedLevel.Set(level)
	configuration := zap.NewProductionConfig()
	configuration.Level = zap.NewAtomicLevelAt(parsedLevel)
	return zap.Must(configuration.Build())
}
