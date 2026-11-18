package controller

import (
	"community/internal/service"
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AgentController struct {
	agentService service.AgentService
}

func (ac *AgentController) Run(c *gin.Context) {
	var req struct {
		Question string `json:"question" binding:"required"`
	}

	var sizeErr *http.MaxBytesError
	if err := BindJSONLimited(c, &req); err != nil {
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体长度超过限制:" + err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合格:" + err.Error()})
	}

	question, err := ValidateText(req.Question, 2000, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question:" + err.Error()})
		return
	}

	body, statusCode, err := ac.agentService.Run(
		c.Request.Context(),
		question,
	)
	if err != nil {
		log.Printf("调用 Agent 失败：%v", err)

		// 请求已被取消时，结束处理。
		if c.Request.Context().Err() != nil {
			return
		}

		status := http.StatusBadGateway
		message := "Agent 服务请求失败"

		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
			message = "等待 Agent 响应超时"
		}

		c.JSON(status, gin.H{"error": message})
		return
	}

	// body 已经是 JSON 字节，直接作为响应正文返回。
	c.Data(statusCode, "application/json; charset=utf-8", body)
}
