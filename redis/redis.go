package redis

import (
	"context" //context 用来在一个请求的整个生命周期（从收到 HTTP 请求到返回响应）中传递超时、取消信号和共享数据。用于超时控制和取消传递
	"fmt"
	"log"

	"github.com/go-redis/redis/v8" //go-redis对Redis的作用类似于GORM对sql的作用

	"community/config"
)

var RDB *redis.Client          //redis.Client是go-redis内置的一个结构体
var Ctx = context.Background() // 是一个空的根 Context,临时替代

func InitRedis() error {
	RDB = redis.NewClient(&redis.Options{ //redis.Options是一个结构体，把这个结构体传给redis.NewClient（）这个函数，这个函数会自动解析结构体里面的每一个值，然后用这些配置创建一个能管理连接、执行命令的 Redis 客户端对象。
		Addr:     config.Cfg.RedisAddr,
		Password: config.Cfg.RedisPassword,
		DB:       0, //使用默认DB
	})
	//测试连接
	_, err := RDB.Ping(Ctx).Result() //Ping 是一个用来测试“对方是否活着、网络是否通的”命令或方法,并且在此次执行命令时自动连接redis
	if err != nil {
		return fmt.Errorf("连接Redis 失败:%w", err)
	}
	log.Printf("redis连接成功")
	return nil
}
