package main

import (
	"community/internal/config"
	"community/internal/controller"
	"community/internal/db"
	"community/internal/middleware"
	"community/internal/redis"
	"community/internal/service"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644) //创建一个日志文件，os.O_CREATE 如果文件不存在，就创建它，os.O_WRONLY 以“只写”模式打开（不能读），os.O_APPEND 写入时追加到文件末尾，不覆盖已有内容，0644 文件权限（Linux 里：所有者可读写，其他人只读）
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, f)) //把日志打印到终端和日志文件
	config.LoadConfig()                         //加载配置

	//连接数据库
	if err := db.InitDB(); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	if err := db.AutoMigrate(); err != nil { //AutoMigrate()是GORM自带对的一个函数，自动把参数里面的指针所指结构体建成数据库对应的表（只增不删），由于数据库连接对象*gorm.DB是在db包里面定义的，所以用db包来调用此函数
		log.Fatal("自动迁移失败", err)
	}

	//连接redis
	if err := redis.InitRedis(); err != nil {
		log.Fatal("Redis初始化失败", err)
	}

	r := gin.Default()
	agentController := controller.AgentController{}
	userController := controller.UserController{}
	postController := controller.PostController{}
	commentController := controller.CommentController{}
	likeController := controller.LikeController{}
	followController := controller.FollowController{}
	feedController := controller.FeedController{}
	hotController := controller.HotController{}
	//公开路由（不需要登陆）
	r.POST("/api/register", userController.Register)
	r.POST("/api/login", userController.Login) //get的本质是向服务器获取信息，而post是提交信息，可能会改变服务器状态， 而登陆会提交密码，还会生成新token，所以是post
	r.GET("/api/posts", postController.GetPostList)
	r.GET("/api/post/:id", postController.GetPostDetail)
	r.GET("/api/posts/:post_id/comments", commentController.GetComments)
	r.GET("/api/posts/:post_id/like-status", middleware.OptionalAuthMiddleware(), likeController.GetLikeStatus)
	r.GET("/api/posts/:post_id/like-count", likeController.GetLikeCount)
	r.GET("/api/posts/hot", hotController.GetHot)

	//需要登陆的路由
	authorized := r.Group("api")                //这个分组下的所有路由，URL 都以 /api 开头,避免重复写
	authorized.Use(middleware.AuthMiddleware()) //给这个分组挂上中间件
	{
		//后续需要的登陆接口都在这里
		authorized.POST("/posts", postController.CreatePost)
		authorized.POST("/comments", commentController.CreateComment)
		authorized.POST("/posts/:post_id/like", likeController.ToggleLike)
		authorized.GET("/users/:user_id/followings", followController.GetFollowings)
		authorized.GET("/users/:user_id/followers", followController.GetFollowers)
		authorized.POST("/users/:user_id/follow", followController.ToggleFollow)
		authorized.GET("/feed", feedController.GetFeed)
		authorized.GET("/admin/refresh-hot", hotController.RefreshHot)
		authorized.GET("/posts/:post_id/ai-summary", postController.GetAISummary) //用ai生成帖子摘要的接口要登陆是因为消耗token要花钱，避免被人随意嚯嚯
		authorized.POST("/agent/run", agentController.Run)
		authorized.GET("/test", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{
				"message": "认证成功",
				"user_id": userID,
			}) //这次返回json是在get请求里的，而登陆和注册返回的json写在controller里
		})

	}

	//定时同步：将点赞数从Redis同步到mysql
	likeService := &service.LikeService{}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			if err := likeService.SyncLikeCountToMySQL(ctx); err != nil {
				log.Println("点赞同步失败", err)
			}
		}
	}()

	//添加后台goroutinue定期刷新热榜
	go func() { //匿名函数
		hotService := service.HotService{}
		ticker := time.NewTicker(10 * time.Minute) //time.NewTicker(10 * time.Minute):创建一个定时器，每 10 分钟发一个信号,ticker:定时器对象，它的 C 通道会每隔 10 分钟收到一个信号
		defer ticker.Stop()                        //停止定时器，释放资源，当函数结束时执行
		for range ticker.C {                       //for range ticker.C	每当定时器发信号时，循环体执行一次
			ctx := context.Background() //创建一个空白的“控制面板”，目前没有超时控制，但保留了将来扩展的能力。
			if err := hotService.RefreshHot(ctx); err != nil {
				log.Println("刷新热榜失败:", err)
			} else {
				log.Println("热榜刷新成功")
			}
		}
	}()

	r.GET("/ping", func(c *gin.Context) { //*gin.Context封装了处理一个http请求需要的一切，是一个结构体，包含了各种路径参数的字段和很多种方法
		c.JSON(200, gin.H{"message": "pong"}) //c.JSON是*gin.Context的一个方法，向客户端返回内容
	})
	addr := fmt.Sprintf(":%s", config.Cfg.AppPort) //用sprintf是为了防止8080前面有空格，需要的是:8080
	r.Run(addr)                                    //监听所有网卡的8080端口，其他电脑可以通过我的 IP 地址访问我的服务，比如电脑的 IP 是 192.168.1.100，手机连同一个 Wi-Fi。本机访问 http://localhost:8080/ping 手机访问 http://192.168.1.100:8080/ping

}
