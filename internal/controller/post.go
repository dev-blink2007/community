package controller

import (
	"community/internal/service"
	"fmt"
	"net/http"

	"context"
	"errors"
	"log"
	"strings"
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
	keyword := strings.TrimSpace(c.Query("keyword"))
	//获取分页参数
	page, size, _, err := ParsePagination(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	posts, total, err := pc.postService.GetPostList(page, size, keyword)
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
	id, err := ParsePositiveID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := pc.postService.GetPostByID(id)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "帖子不存在",
			})
			return
		}

		log.Printf("查询帖子失败，post_id=%d，错误：%v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询帖子失败",
		})
		return
	} //把帖子不存在和查询失败的错误分开

	c.JSON(http.StatusOK, gin.H{
		"data": post,
	})
}

// GetAISummary AI生成帖子摘要
func (pc *PostController) GetAISummary(c *gin.Context) {
	//获取帖子ID
	id, err := ParsePositiveID(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post, err := pc.postService.GetPostByIDWithLock(c.Request.Context(), id)
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
	content, token, err := service.CallDeepSeekAPI(ctx, prompt, uint(id))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"content": content,
		"token":   token,
	})
}
