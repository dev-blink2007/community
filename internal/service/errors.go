package service

import "errors"

var ErrUsernameConflict = errors.New("用户名已存在")
var ErrInvalidCredentials = errors.New("用户名或密码错误")
var ErrPostNotFound = errors.New("帖子不存在")
