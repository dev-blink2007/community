package controller

import (
	"community/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FeedController struct {
	feedService service.FeedService
}

func (fc *FeedController) GetFeed(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	userID := userIDVal.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	posts, total, err := fc.feedService.GetFeedWithCache(c.Request.Context(), userID, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Feed失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  posts,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
