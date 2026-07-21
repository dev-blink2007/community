package service

import (
	"community/db"
	"community/model"
	"community/redis"
	"context"
	"fmt"
	"time"
)

type FeedService struct{}

// GetFeed 获取关注用户的帖子列表
func (s *FeedService) GetFeed(userID uint, page, size int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	//先查当前用户关注了那些人
	var followings []model.Follow
	if err := db.DB.Where("follower_id = ?", userID).Find(&followings).Error; err != nil {
		return nil, 0, err
	}

	//如果没关注任何人直接返回空列表
	if len(followings) == 0 {
		return []model.Post{}, 0, nil
	}

	//提取关注人的ID列表
	followingIDs := make([]uint, 0, len(followings))
	for _, f := range followings {
		followingIDs = append(followingIDs, f.FollowingID)
	}

	//查这些人的帖子，按时间倒序
	offset := (page - 1) * size
	query := db.DB.Model(&model.Post{}).Where("user_id IN ?", followingIDs).Order("created_at DESC") //Model(&model.Post{})这里是显式指定表名，告诉gorm对按个表进行操作
	//query不存数据，他只是一个sql语句构建器，可以看作只是存的sql语句，下面每个分支再具体操作

	//统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	//分页查询，预加载用户信息
	if err := query.Preload("User").Offset(offset).Limit(size).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil //这里查询到的帖子就是按时间顺序排列的，须有关注者发布的所有帖子混在一起的大杂烩，这就是feed流

}

// 辅助：根据ID列表查帖子
func (s *FeedService) getPostByIDs(ctx context.Context, idStrs []string) ([]model.Post, error) {
	var ids []uint
	for _, s := range idStrs {
		var id uint
		fmt.Sscanf(s, "%d", &id)
		ids = append(ids, id)
	}

	var posts []model.Post
	if err := db.DB.Where("id IN ?", ids).Preload("User").Find(&posts).Error; err != nil {
		return nil, err
	}

	//按缓存顺序排序
	postMap := make(map[uint]model.Post)
	for _, p := range posts {
		postMap[p.ID] = p
	}
	ordered := make([]model.Post, 0, len(ids))
	for _, id := range ids {
		if p, ok := postMap[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

// 从缓存中获取feed
func (s *FeedService) GetFeedWithCache(ctx context.Context, userID uint, page, size int) ([]model.Post, int64, error) {
	cacheKey := fmt.Sprintf("feed:%d", userID)
	countKey := fmt.Sprintf("feed:count:%d", userID)
	//尝试从Redis取
	total, err := redis.RDB.Get(ctx, countKey).Int64()
	if err == nil {
		postIDs, err := redis.RDB.LRange(ctx, cacheKey, int64((page-1)*size), int64(page*size-1)).Result()
		if err == nil && len(postIDs) > 0 {
			posts, err := s.getPostByIDs(ctx, postIDs)
			if err != nil {
				return nil, 0, err
			}
			return posts, total, nil
		}
	}

	//缓存未命中，从mysql中查询
	posts, totalFromDB, err := s.GetFeed(userID, page, size)
	if err != nil {
		return nil, 0, err
	}

	//缓存未命中时异步写入缓存
	go func() {
		for i := 0; i < 100 && i < len(posts); i++ {
			redis.RDB.RPush(ctx, cacheKey, posts[i].ID)
		}
		redis.RDB.Expire(ctx, cacheKey, 5*time.Minute)
		redis.RDB.Set(ctx, countKey, totalFromDB, 1*time.Hour)
	}()
	return posts, totalFromDB, nil
}
