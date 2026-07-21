package model

import (
	"time"
)

type Post struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"size:200;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LikeCount int64     `json:"like_count" gorm:"default:0"`

	//关联关系（非数据库字段，GORM自动处理）
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
