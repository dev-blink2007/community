package middleware

import (
	"net/http"
	"strings"

	"community/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT验证中间件
func AuthMiddleware() gin.HandlerFunc { //gin.HandlerFunc是一个接收*gin.Context不返回的函数类型,可以视作一个接口规范
	return func(c *gin.Context) {
		//从Header里面获取token      //token在Header里是标准做法，不会被日志记录且所有请求都能带，（get请求没有body）
		authHeader := c.GetHeader("Authorization") //c.GetHeader("Authorization")是Gin框架提供的一个方法，用于从Header中获取指定key的值,token在authoriza里
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证token"})
			c.Abort() //c.Abort() 的作用是：停止当前请求的后续处理链，不再执行后续的中间件或路由处理函数
			return    //return的作用是停止当前函数，c.Abort()的作用是停止当前请求链
		}

		//authoriza里面的格式是:Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2) //按空格把authorHeader分成最多两段
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token格式错误"})
			c.Abort()
			return
		}
		tokenString := parts[1]
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}
		//把用户id存进上下文，后续接口可以直接取
		c.Set("user_id", claims.UserID) //c.Set(key, value) 是在 Gin 的上下文里存储一个键值对，这个值只在当前请求的生命周期里存在。
		//Gin 为了通用性，把存进去的值变成了 interface{} 类型
		c.Next()
	}

}
