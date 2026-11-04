package problem

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteUsesProblemDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/examples", nil)
	Write(context, Validation("参数错误"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码为 %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != MediaType {
		t.Fatalf("内容类型为 %q", recorder.Header().Get("Content-Type"))
	}
}
