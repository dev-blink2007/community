package config

import (
	"log"
	"os" //读取系统环境里的环境变量.也就是只能读已经存在的环境变量

	//"stronv"//对字符串进行操作

	"github.com/joho/godotenv" //从.env文件中读取自己写的配置，然后将内容搬运到系统环境变量里，也就是os能读的环境变量里，密码这些不会被传到git上，实现配置与代码分离
)

type AppConfig struct {
	AppPort        string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	RedisAddr      string
	RedisPassword  string
	JWTSecret      string
	DeepSeekAPIKey string
}

var Cfg *AppConfig //指向AppConfig结构体的指针

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("未找到.env文件,将使用系统环境变量") //开发阶段暂时将报错日志打印到终端，等到最后项目快结束打磨的时候再加到日志文件中去
	}
	Cfg = &AppConfig{
		AppPort:        getEnv("APP_PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "community"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		JWTSecret:      getEnv("JWT_SECRET", "default-secret"),
		DeepSeekAPIKey: getEnv("DEEPSEEK_API_KEY", ""),
	}
}
func getEnv(key, defaultVaiue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVaiue
}
