package controller

import (
	"community/internal/service"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AgentController struct {
	agentService service.AgentService
}

func (ac *AgentController) Run(c *gin.Context) {
	var req struct {
		Question string `json:"question" binding:"required,max=2000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "question 必须是长度不超过2000的非空字符串",
		})
		return
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "问题不能为空或只有空白字符",
		})
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
