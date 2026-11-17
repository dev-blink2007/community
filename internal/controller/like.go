package controller

import (
	"community/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LikeController struct {
	likeService service.LikeService
}

// ToggleLike 点赞/取消点赞
func (lc *LikeController) ToggleLike(c *gin.Context) {
	userIDval, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	userID := userIDval.(uint)

	postID, err := ParsePositiveID(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	liked, err := lc.likeService.ToggleLike(c.Request.Context(), userID, postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	message := "点赞成功"
	if !liked {
		message = "取消点赞成功"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"liked":   liked,
	})
}

// GetLikeStatus 获取点赞状态
func (lc *LikeController) GetLikeStatus(c *gin.Context) {
	postID, err := ParsePositiveID(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//不登录则默认未点赞
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{"liked": false})
		return
	}
	userID := userIDVal.(uint)

	liked, err := lc.likeService.GetLikeStatus(userID, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

// GetLikeCount 获取点赞总数
func (lc *LikeController) GetLikeCount(c *gin.Context) {
	postID, err := ParsePositiveID(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := lc.likeService.GetLikeCountWithCache(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}
