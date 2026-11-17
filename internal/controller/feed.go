package controller

import (
	"community/internal/service"
	"net/http"

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

	page, size, _, err := ParsePagination(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
