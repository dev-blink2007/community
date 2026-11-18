package controller

import (
	"community/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CommentController struct {
	commentService service.CommentService
}

// CreateComment 发布评论
func (cc *CommentController) CreateComment(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	userID := userIDVal.(uint)

	var req struct {
		PostID  uint   `json:"post_id" binding:"required"`
		Content string `json:"content" binding:"required,min=1"`
	}
	var sizeErr *http.MaxBytesError
	if err := BindJSONLimited(c, &req); err != nil {
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体长度超过限制:" + err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合格:" + err.Error()})
	}
	content, err := ValidateText(req.Content, 2000, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论:" + err.Error()})
		return
	}
	comment, err := cc.commentService.CreateComment(userID, req.PostID, content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "评论成功",
		"comment": comment,
	})
}

// GetComments 获取帖子的评论列表
func (cc *CommentController) GetComments(c *gin.Context) {
	postID, err := ParsePositiveID(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comments, err := cc.commentService.GetCommentsByPostID(postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": comments,
	})
}
