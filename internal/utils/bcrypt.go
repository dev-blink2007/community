package utils

import (
	"golang.org/x/crypto/bcrypt" //go官方的密码加密库，自带盐值，哈希成本可调整，抗暴力破解能力强
)

// HashPassword加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost) //bcrypt.DefaultCost是一个常量，值为10，表示哈希的计算成本（密码哈希2^cost次），可以自己定义，用数字替换掉这个常量
	//[]byte(password)是类型转换，将里面的字符串转换为字节切片类型
	//bcrypt.GenerateFromPassword是加密密码的函数，自带盐值，盐值是随机的但调用一次，不管反复哈希多少次都用的同一个盐值，哈希次数又第二个参数决定，参数值越大，哈希成本越高，用户端等待时间增加微乎其微，但是攻击者上亿次的暴力破解的成本会呈指数级增长
	return string(bytes), err
}

// CheckPassword验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) //用哈希值里面明文的盐值计算用户输入密码，验证哈希后与给定的字符串是否相等，相等则密码输入正确，只用尝试一次，而攻击者需要尝试上亿次
	return err == nil
}
