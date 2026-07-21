package service

import (
	"community/db"
	"community/model"
	"errors"

	"gorm.io/gorm"
)

type FollowService struct{}

// ToggleFollow 关注/取消关注
func (s *FollowService) ToggleFollow(followerID, followingID uint) (bool, error) {
	if followerID == followingID {
		return false, errors.New("不能关注自己")
	}

	var follow model.Follow
	err := db.DB.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&follow).Error

	if err == nil {
		//已关注--取关
		if err := db.DB.Delete(&follow).Error; err != nil {
			return false, err
		}
		return false, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		//未关注==关注
		follow = model.Follow{
			FollowerID:  followerID,
			FollowingID: followingID,
		}
		if err := db.DB.Create(&follow).Error; err != nil {
			return false, err
		}
		return true, nil
	}
	return false, err
}

// GetFollowings 获得关注列表
func (s *FollowService) GetFollowings(userID uint) ([]model.User, error) {
	var followings []model.User
	err := db.DB.Table("users").Joins("INNER JOIN follows ON users.id = follows.following_id").Where("follows.follower_id = ?", userID).Find(&followings).Error
	//主表是user表，因为我们要得到的是用户数据列表而不是关注信息列表，然后将followingID等于userID时的所有follows数据都拼接到这个ID对应的user数据后面，，如果一个userID对应多条关注数据，user表会纵向拉长，相同的用户信息不同的关注信息生成新数据
	//虽然followerID等于目标数据的信息分布在该表各处，但是建立了联合索引，数据库会走B+数索引，先根据前缀定位，再在定位到的分支里按第二个字段（following_id）继续定位。
	return followings, err
}

// GetFollowers 获取粉丝列表
func (s *FollowService) GetFollowers(userID uint) ([]model.User, error) {
	var followers []model.User
	err := db.DB.Table("users").Joins("INNER JOIN follows ON users.id = follows.following_id").Where("follows.following_id = ?", userID).Find(&followers).Error
	//这里没用上联合索引，但是表顺序是按照following来的
	return followers, err
}
