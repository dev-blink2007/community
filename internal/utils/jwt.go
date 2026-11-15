package utils

import (
	"community/internal/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID               uint `json:"user_id"`
	jwt.RegisteredClaims      //（匿名字段）一个结构体，里面包含签发时间，过期时间，签发人,这个匿名字段的名字叫RegisteredClaims，匿名字段的好处就是可以直接claims.ExpiresAt而不是每次都得claims.RegisteredClaims.ExpiresAt
}

// 生成jwt
func GenerateToken(userID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), //过期时间，jwt.NewNumericDate()把Go的time.Time类型转换成JWT 标准要求时间的 NumericDate 格式
			IssuedAt:  jwt.NewNumericDate(time.Now()),                     //签发时间
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) //jwt.SigningMethodES256是一个常数，指定要生成的签名所用的算法，用来防止Header、Payload被篡改，确保jwt的完整性
	//jwt.NewWithClaims把要放进JWT的数据（claims）和签名算法（SigningMethodHS256）打包在一起，生成一个“待签名”的 token 对象，相当于填支票内容
	return token.SignedString([]byte(config.Cfg.JWTSecret)) //先用指定的密钥（JWTSecret）对前面打包好的内容进行签名，然后把 Header、Payload、Signature 三部分用 . 连接成一个字符串
}

// 解析jwt
func ParseToken(tokenstring string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenstring, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		secret := []byte(config.Cfg.JWTSecret)
		if config.Cfg.JWTSecret == "default-secret" {
			return nil, fmt.Errorf("密钥不应该使用默认值")
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	//该函数：1.把 JWT 字符串切成三段：header、payload、signature，分别做 base64 解码。2.把 payload 的 JSON 填进 &Claims{}。这就是为什么第二个参数要传一个指针：填数据需要能改到那个结构体。3.调用 keyFunc 拿到密钥，验证签名。签名不对ParseWithClaims就返回 error。4.返回一个 *jwt.Token，里面装着解析后的 header、claims、签名等信息。
	//这个keyfunc函数作用就是返回验证签名需要的密钥，供ParseWithClaims作验证使用，当有多个密钥的时候这个函数还可以通过传入的token里面的header字段内容选择密钥
	if err != nil { //如果密钥不对 jwt.ParseWithClaims这个函数就会报错
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid { //token.Claims.(*Claims)是类型断言，语法格式为：接口变量.(具体类型)，将接口类型转换为具体类型，检验token.Claims这个接口是否为*Claims类型，是则将数据填入结构体，且ok为true，token.Claims是token里面的一个字段，它是一个实现了valid的接口。你传进去的 &Claims{} 会被填满后赋值给这个接口。
		//valid是判断这个token是否还有效的方法
		return nil, errors.New("invalid token claims")
	}
	if claims.UserID <= 0 {
		return nil, errors.New("token 中的用户 ID 无效")
	}
	return claims, nil
}
