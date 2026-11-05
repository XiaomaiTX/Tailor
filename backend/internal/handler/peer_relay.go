package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/XiaomaiTX/go-react-starter/internal/middleware"
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterPeerRelayRoutes(engine *gin.Engine, peer *service.PeerRelayService) {
	group := engine.Group("/api/v1")
	group.GET("/relay/me", func(c *gin.Context) {
		value, err := peer.Me(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
	group.GET("/relay/dashboard", func(c *gin.Context) {
		value, err := peer.Dashboard(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
	group.GET("/settings", func(c *gin.Context) {
		value, err := peer.Settings(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
	group.PUT("/settings", func(c *gin.Context) {
		var body service.SettingsInput
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.UpdateSettings(c.Request.Context(), middleware.Principal(c.Request.Context()), body)
		writePeerResponse(c, value, err)
	})
	group.GET("/devices", func(c *gin.Context) {
		search := strings.TrimSpace(c.Query("search"))
		if len([]rune(search)) > 120 {
			writePeerResponse(c, nil, service.ErrValidation)
			return
		}
		value, err := peer.Devices(c.Request.Context(), middleware.Principal(c.Request.Context()), search)
		writePeerResponse(c, value, err)
	})
	group.GET("/relays", func(c *gin.Context) {
		includeAll := false
		if raw := c.Query("includeAll"); raw != "" {
			parsed, err := strconv.ParseBool(raw)
			if err != nil {
				writePeerResponse(c, nil, service.ErrValidation)
				return
			}
			includeAll = parsed
		}
		value, err := peer.Relays(c.Request.Context(), middleware.Principal(c.Request.Context()), includeAll)
		writePeerResponse(c, value, err)
	})
	group.POST("/relays", func(c *gin.Context) {
		var body struct {
			NodeID string `json:"nodeId" binding:"required"`
			service.RelayInput
		}
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.RegisterRelay(c.Request.Context(), middleware.Principal(c.Request.Context()), body.NodeID, body.RelayInput)
		if err != nil {
			writePeerResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusCreated, value)
	})
	group.PUT("/relays/:nodeId", func(c *gin.Context) {
		var body service.RelayInput
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.UpdateRelay(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("nodeId"), body)
		writePeerResponse(c, value, err)
	})
	group.DELETE("/relays/:nodeId", func(c *gin.Context) {
		if err := peer.CancelRelay(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("nodeId")); err != nil {
			writePeerResponse(c, nil, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	group.GET("/applications", func(c *gin.Context) {
		value, err := peer.Applications(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Query("status"))
		writePeerResponse(c, value, err)
	})
	group.POST("/applications", func(c *gin.Context) {
		var body service.ApplicationInput
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.CreateApplication(c.Request.Context(), middleware.Principal(c.Request.Context()), body)
		if err != nil {
			writePeerResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusCreated, value)
	})
	group.POST("/applications/:id/cancel", func(c *gin.Context) {
		if err := peer.CancelApplication(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("id")); err != nil {
			writePeerResponse(c, nil, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	group.POST("/applications/:id/review", func(c *gin.Context) {
		var body struct {
			Decision string              `json:"decision" binding:"required"`
			Note     string              `json:"note"`
			Relay    *service.RelayInput `json:"relay"`
		}
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.ReviewApplication(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("id"), body.Decision, body.Note, body.Relay)
		writePeerResponse(c, value, err)
	})
	group.GET("/graph/all", func(c *gin.Context) {
		value, err := peer.CanvasGraph(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
	group.GET("/graph", func(c *gin.Context) {
		sourceNodeID := strings.TrimSpace(c.Query("sourceNodeId"))
		if sourceNodeID == "" {
			writePeerResponse(c, nil, service.ErrValidation)
			return
		}
		value, err := peer.Graph(c.Request.Context(), middleware.Principal(c.Request.Context()), sourceNodeID)
		writePeerResponse(c, value, err)
	})
	group.PUT("/assignments", func(c *gin.Context) {
		var body service.AssignmentInput
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.UpdateAssignments(c.Request.Context(), middleware.Principal(c.Request.Context()), body)
		writePeerResponse(c, value, err)
	})
	group.PUT("/assignments/batch", func(c *gin.Context) {
		var body service.BatchAssignmentInput
		if !bindPeerJSON(c, &body) {
			return
		}
		value, err := peer.UpdateAssignmentsBatch(c.Request.Context(), middleware.Principal(c.Request.Context()), body)
		writePeerResponse(c, value, err)
	})
	group.POST("/sync", func(c *gin.Context) {
		if err := peer.SyncNow(c.Request.Context(), middleware.Principal(c.Request.Context())); err != nil {
			writePeerResponse(c, nil, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	group.GET("/policy/revisions", func(c *gin.Context) {
		value, err := peer.Revisions(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
	group.GET("/policy/revisions/:id", func(c *gin.Context) {
		value, err := peer.RevisionDetail(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("id"))
		writePeerResponse(c, value, err)
	})
	group.POST("/policy/revisions/:id/rollback", func(c *gin.Context) {
		value, err := peer.Rollback(c.Request.Context(), middleware.Principal(c.Request.Context()), c.Param("id"))
		writePeerResponse(c, value, err)
	})
	group.GET("/audit", func(c *gin.Context) {
		value, err := peer.Audits(c.Request.Context(), middleware.Principal(c.Request.Context()))
		writePeerResponse(c, value, err)
	})
}

func bindPeerJSON(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writePeerResponse(c, nil, service.ErrValidation)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writePeerResponse(c, nil, service.ErrValidation)
		return false
	}
	return true
}

func writePeerResponse(c *gin.Context, value any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, value)
		return
	}
	switch {
	case errors.Is(err, service.ErrValidation):
		problem.Write(c, problem.Validation("请检查请求参数或内容"))
	case errors.Is(err, repository.ErrNotFound):
		problem.Write(c, problem.NotFound())
	case errors.Is(err, service.ErrForbidden):
		problem.Write(c, problem.Forbidden())
	case errors.Is(err, service.ErrConflict):
		problem.Write(c, problem.Conflict(err.Error()))
	case errors.Is(err, service.ErrUnavailable):
		problem.Write(c, problem.Unavailable("Headscale 或必要服务暂时不可用"))
	case errors.Is(err, service.ErrPolicyWrites):
		problem.Write(c, problem.Unavailable("Headscale Policy 写入未启用"))
	default:
		problem.Write(c, problem.Internal())
	}
}
