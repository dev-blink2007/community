package model

import "time"

type Follow struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	FollowerID  uint      `json:"follower_id" gorm:"not null;index:idx_follow_pair,unique"`
	FollowingID uint      `json:"following_id" gorm:"not null;index:idx_follow_pair,unique"` //通过两个ID生成新的联合索引(follower_id, following_id)
	CreatedAt   time.Time `json:"created_at"`
}
