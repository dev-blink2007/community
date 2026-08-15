package controller

import (
	"community/service"
	"fmt"
	"net/http"
	"strconv"

	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type PostController struct {
	postService service.PostService
}

// CreatePost 发布帖子（需要登陆）
func (pc *PostController) CreatePost(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	userID := userIDVal.(uint) //把接口类型转换成uint

	var req struct {
		Title   string `json:"title" binding:"required,min=1,max=200"`
		Content string `json:"content" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误" + err.Error()})
		return
	}

	post, err := pc.postService.CreatePost(userID, req.Title, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "发布成功",
		"post":    post,
	})
}

// GetPostList 获取帖子列表（分页）
func (pc *PostController) GetPostList(c *gin.Context) {
	//获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1")) //获取URL里面page的值，没有就用默认值1
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	posts, total, err := pc.postService.GetPostList(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取帖子列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  posts,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetPostDetail 获取帖子详情
func (pc *PostController) GetPostDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	post, err := pc.postService.GetPostByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": post,
	})
}

// GetAISummary AI生成帖子摘要
func (pc *PostController) GetAISummary(c *gin.Context) {
	//获取帖子ID
	idstr := c.Param("post_id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}
	post, err := pc.postService.GetPostByIDWithLock(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	prompt := fmt.Sprintf("标题：%s 内容： %s", post.Title, post.Content)
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	timeout, _ := time.ParseDuration("30s")
	ctx, cancel = context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	content, token, err := service.CallDeepSeekAPI(ctx, prompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"content": content,
		"token":   token,
	})
}
