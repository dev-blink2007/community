package model

import (
	"time"
)

type Like struct {
	ID        uint      `json:"id" gorm:"primaryKey"` //点赞记录的编号
	UserID    uint      `json:"user_id" gorm:"not null;index:idx_user_post,unique"`
	PostID    uint      `json:"post_id" gorm:"not null;index:idx_user_post,unique"` //创建联合索引，每个帖子每个用户只能点一次赞
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
