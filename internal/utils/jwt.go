package utils

import (
	"community/internal/config"
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
	token, err := jwt.ParseWithClaims(tokenstring, &Claims{}, func(token *jwt.Token) (interface{}, error) { //&Claims{}这一步有两个意思，创建Claims{}的实例，再取他（实例）的地址
		//jwt.ParseWithClaims是解析函数，返回 jwt.Token 结构体，判断解析出来的信息能否填入Claims{}里，能就填入，有三个参数tokenString string，要解析的JWT字符串；claims Claims，存放解析结果的结构体；keyFunc Keyfunc，回调函数，返回验证签名的密钥，且该函数的参数，返回值类型是被定义好的
		//该函数第二个参数是一个接口类型，实现Valid（）的接口类型，jwt.RegisteredClaims写好（实现）了这个方法，而Claims{}继承了这个方法,所以这个结构体的指针类型可以视作该接口类型
		return []byte(config.Cfg.JWTSecret), nil
	})
	if err != nil { //如果密钥不对 jwt.ParseWithClaims这个函数就会报错
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid { //token.Claims.(*Claims)是类型断言，语法格式为：接口变量.(具体类型)，将接口类型转换为具体类型，检验token.Claims这个接口是否为*Claims类型，是则将数据填入结构体，且ok为true，token.Claims是token里面的一个字段，它是一个实现了valid的接口。你传进去的 &Claims{} 会被填满后赋值给这个接口。
		//valid是判断这个token是否还有效的方法
		return claims, nil
	}
	return nil, err
}
