package problem

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const MediaType = "application/problem+json"

type Details struct {
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Status   int     `json:"status"`
	Detail   *string `json:"detail,omitempty"`
	Instance *string `json:"instance,omitempty"`
}

func New(status int, typeName, title string, detail *string) Details {
	return Details{Type: "urn:go-react-starter:problem:" + typeName, Title: title, Status: status, Detail: detail}
}

func Validation(detail string) Details {
	return New(http.StatusBadRequest, "validation", "请求不合法", &detail)
}
func NotFound() Details     { return New(http.StatusNotFound, "not-found", "资源不存在", nil) }
func Unauthorized() Details { return New(http.StatusUnauthorized, "unauthorized", "需要登录", nil) }
func Forbidden() Details {
	return New(http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
}
func Conflict(detail string) Details {
	return New(http.StatusConflict, "conflict", "数据已变化", &detail)
}
func Unavailable(detail string) Details {
	return New(http.StatusServiceUnavailable, "unavailable", "服务暂不可用", &detail)
}
func Internal() Details {
	return New(http.StatusInternalServerError, "internal", "服务内部错误", nil)
}

func Write(c *gin.Context, value Details) {
	value.Instance = pointer(c.Request.URL.Path)
	c.Header("Content-Type", MediaType)
	c.JSON(value.Status, value)
}

func pointer(value string) *string { return &value }
