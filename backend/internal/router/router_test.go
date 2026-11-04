package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/handler"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"go.uber.org/zap"
)

type emptyExamples struct{}

func (emptyExamples) List(context.Context, int, int) (service.ExamplePage, error) {
	return service.ExamplePage{}, nil
}
func (emptyExamples) Get(context.Context, int64) (model.Example, error) { return model.Example{}, nil }
func (emptyExamples) Create(context.Context, service.ExampleInput) (model.Example, error) {
	return model.Example{}, nil
}
func (emptyExamples) Update(context.Context, int64, service.ExampleInput) (model.Example, error) {
	return model.Example{}, nil
}
func (emptyExamples) Delete(context.Context, int64) error { return nil }

func TestLiveHealthIncludesBaseMiddlewares(t *testing.T) {
	api := handler.NewAPI(emptyExamples{}, func(context.Context) error { return nil }, func(context.Context) error { return nil })
	engine := New(api, config.Config{CORSOrigins: []string{"http://localhost:5173"}}, zap.NewNop())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码为 %d", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("缺少请求 ID")
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("缺少安全响应头")
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatal("CORS 未生效")
	}
}
