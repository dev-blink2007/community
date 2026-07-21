package controller

import (
	"community/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type HotController struct {
	hotService service.HotService
}

// GetHot 拉取热榜
func (hc *HotController) GetHot(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	posts, err := hc.hotService.GetHotPost(c.Request.Context(), limit) //c.Request是一个HTTP标准请求体，c.Request.Context()就是从请求体中取出 context.Context
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取热榜失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  posts,
		"total": len(posts),
	})
}

// RefreshHot 手动刷新热榜
func (hc *HotController) RefreshHot(c *gin.Context) {
	err := hc.hotService.RefreshHot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "刷新成功"})
}
