package db

import (
	"database/sql"
	"fmt"
	"log" //在程序启动时加载日志，避免出现配置未加载问题
	"time"

	"gorm.io/driver/mysql" //告诉GORM翻译的是mysql的语言
	"gorm.io/gorm"         //将结构体翻译成数据库表，并将db.Create(&user)这样的操作翻译成数据库语言
	"gorm.io/gorm/logger"  //将sql的操作翻译成日志语言

	"community/internal/config"
	"community/internal/model"
)

var DB *gorm.DB //DB是GORM里面写好的一个结构体

// InitDB初始化数据库连接
func InitDB() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		config.Cfg.DBUser,
		config.Cfg.DBPassword,
		config.Cfg.DBHost,
		config.Cfg.DBPort,
		config.Cfg.DBName, //结尾最后留一个逗号，方便增删行
	)
	//root:123456@tcp(localhost:3306)/community?charset=utf8mb4&parseTime=True&loc=Local
	// 	root:123456	       用户名:密码	    身份验证
	// @	               分隔符	       把“身份”和“地址”分开
	// tcp(localhost:3306) 协议(地址:端口)	MySQL 服务器在哪
	// /community	       /数据库名	   连接哪个库
	// ?	               分隔符	       后面是参数
	// charset=utf8mb4	   参数1	       支持中文和 emoji
	// parseTime=True	   参数2	       自动把 MySQL 的 datetime 转成 Go 的 time.Time
	// loc=Local	       参数3	       使用本地时区
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{ //gorm.Open(...) GORM 的入口函数 创建数据库连接对象gorm.Open 的第一个参数是 Dialector 接口，mysql.Open(dsn) 返回的就是这个接口 解耦。GORM 核心不依赖具体的数据库实现。你想换成 PostgreSQL，只需要把 mysql.Open(dsn) 换成 postgres.Open(dsn)，其他地方完全不用改。这叫面向接口编程 &gorm.Config{ Logger: ... } 配置GORM的行为.这里只配置了logger字段，将其变成一个能打印sql操作的日志器，然后将gorm.Config这个指针传给GORM
		Logger: logger.Default.LogMode(logger.Info), //logger.Default GORM 自带的默认日志器   .LogMode(logger.Info) 设置日志级别为 Info（打印所有 SQL）
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败:%w", err) //如果报错就会直接退出函数
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接池失败:%w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	log.Println("MySQL连接成功")
	return nil
}

// Status 返回数据库连接池当前状态。
func Status() (sql.DBStats, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return sql.DBStats{}, err
	}
	return sqlDB.Stats(), nil
}

// AutoMigrate 自动创建/更新表（后面加了新 Model 就往里加）
func AutoMigrate() error {
	err := DB.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Like{},
		&model.Follow{},
	)
	if err != nil {
		return err
	}
	log.Println("数据库表迁移完成")
	return nil
}
