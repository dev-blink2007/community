package controller

import (
	"community/internal/service"
	"community/internal/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService service.UserService
}

// Register注册接口
func (uc *UserController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	var sizeErr *http.MaxBytesError
	if err := BindJSONLimited(c, &req); err != nil {
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体长度超过限制:" + err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合格:" + err.Error()})
	}
	password, err := ValidatePassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码:" + err.Error()})
		return
	}

	user, err := uc.userService.Register(req.Username, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成JWT
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"user_id": user.ID,
		"token":   token,
	})
}

// 登陆接口
func (uc *UserController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var sizeErr *http.MaxBytesError
	if err := BindJSONLimited(c, &req); err != nil {
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体长度超过限制:" + err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不合格:" + err.Error()})
	}
	password, err := ValidatePassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码:" + err.Error()})
		return
	}
	user, err := uc.userService.Login(req.Username, password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	//生成新token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登陆成功",
		"user_id": user.ID,
		"token":   token,
	})
}
