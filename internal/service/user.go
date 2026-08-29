package service

import (
	"community/internal/db"
	"community/internal/model"
	"community/internal/utils"
	"errors"

	"gorm.io/gorm"
)

type UserService struct{}

// Register注册新用户
func (s *UserService) Register(username, password string) (*model.User, error) {
	//检查用户是否存在
	var existingUser model.User
	if err := db.DB.First(&existingUser, "username=?", username).Error; err == nil { //查询成功则err为空
		//existingUser是User类型，所以数据库查users表，并且把查到的数据填在&existingUser对应的地址内
		return nil, errors.New("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) { //有其他错误，但不是记录不存在的问题
		return nil, err
	}
	//加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	//创建用户
	user := model.User{
		Username: username,
		Password: hashedPassword,
	}
	if err := db.DB.Create(&user).Error; err != nil { //这里做了两件事，在数据库中创建一条新数据，并且id会自增赋值给user
		return nil, err
	}
	return &user, nil //传指针是为了防止值结构体过大，返回效率更高
}

// login用户登陆
func (s *UserService) Login(username, password string) (*model.User, error) {
	var user model.User
	//查询用户
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误") //两种错误混合描述，防止攻击者知道用户名存在后定向爆破
		}
		return nil, err
	}

	//验证密码
	if !utils.CheckPassword(password, user.Password) {
		return nil, errors.New("用户名或密码错误")
	}
	return &user, nil
}
