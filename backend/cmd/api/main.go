package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/handler"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"github.com/XiaomaiTX/go-react-starter/internal/router"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
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

	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o750); err != nil {
		logger.Fatal("创建 SQLite 数据库目录失败", zap.Error(err))
	}
	database, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		logger.Fatal("打开 SQLite 数据库失败", zap.Error(err))
	}
	if err := database.AutoMigrate(&model.Example{}); err != nil {
		logger.Fatal("初始化 SQLite 表结构失败", zap.Error(err))
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		logger.Fatal("获取 SQLite 连接池失败", zap.Error(err))
	}
	defer sqlDatabase.Close() //nolint:errcheck

	exampleRepository := repository.NewExampleRepository(database)
	exampleService := service.NewExampleService(exampleRepository)
	api := handler.NewAPI(exampleService, sqlDatabase.PingContext)
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
