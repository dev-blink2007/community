package model

import (
	"time"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`                         //`json:"id" gorm:"primaryKey"` 是结构体的标签，定义的时候不需要导入包，只有运行时需要对应的包来读取
	Username  string    `json:"username" gorm:"uniqueIndex;size:50;not null"` //uniqueIndex,唯一索引,这个字段的值在整个表里不能重复（比如用户名不能相同）size:50,字段长度,在 MySQL 里对应 VARCHAR(50)not null,不能为空,插入数据时这个字段必须有值
	Password  string    `json:"-" gorm:"not null"`                            //  json:"-" 表示JSON序列化时忽略
	CreatedAt time.Time `json:"created_at"`                                   //time.Time 是一个结构体，内部保存了秒数 + 纳秒数 + 时区信息,还有自带的方法
	UpdatedAt time.Time `json:"updated_at"`
}
