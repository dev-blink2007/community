package controller

import (
	"community/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FollowController struct {
	followService service.FollowService
}

// ToggleFollow 关注/取消关注
func (fc *FollowController) ToggleFollow(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists { //防御性编程，防止中间件崩溃导致用户未登陆就进来了，预防接下来的类型断言panic
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	userID := userIDVal.(uint)

	targetID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	followed, err := fc.followService.ToggleFollow(userID, uint(targetID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := "关注成功"
	if !followed {
		msg = "取消关注成功"
	}
	c.JSON(http.StatusOK, gin.H{
		"message":  msg,
		"followed": followed,
	})
}

func (fc *FollowController) GetFollowings(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	users, err := fc.followService.GetFollowings(uint(targetID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": users,
	})
}

func (fc *FollowController) GetFollowers(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	users, err := fc.followService.GetFollowers(uint(targetID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": users,
	})
}
