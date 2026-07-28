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

const (
	HotCachePages = 10
	MaxCacheItems = 500
)

func (s *FeedService) GetFeedWithCache(ctx context.Context, userID uint, page, size int) ([]model.Post, int64, error) {
	cachekey := fmt.Sprintf("feed:%d", userID)
	countkey := fmt.Sprintf("feed:count:%d", userID)
	if page > HotCachePages {
		posts, total, err := s.GetFeed(userID, page, size)
		if err != nil {
			return nil, 0, err
		}
		return posts, total, nil
	}

	pipe := redis.RDB.Pipeline()
	start := int64((page - 1) * size)
	end := int64(page*size - 1)
	idcmd := pipe.LRange(ctx, cachekey, start, end)
	totalcmd := pipe.Get(ctx, countkey)
	pipe.Exec(ctx)

	postIDs, err1 := idcmd.Result()
	total, err2 := totalcmd.Int64()
	if err1 == nil && err2 == nil && len(postIDs) > 0 {
		posts, err := s.getPostByIDs(ctx, postIDs)
		if err != nil {
			return nil, 0, err
		}
		return posts, total, nil
	}

	posts, count, err := s.GetFeed(userID, page, size)
	if err != nil {
		return nil, 0, err
	}
	go s.buildHotCache(context.Background(), userID, size)
	return posts, count, nil

}

func (s *FeedService) buildHotCache(ctx context.Context, userID uint, size int) {
	cachekey := fmt.Sprintf("feed:%d", userID)
	countkey := fmt.Sprintf("feed:count:%d", userID)

	cachelimit := HotCachePages * size
	if cachelimit > MaxCacheItems {
		cachelimit = MaxCacheItems
	}

	posts, total, err := s.GetFeed(userID, 1, cachelimit)
	if err != nil || len(posts) == 0 {
		return
	}
	pipe := redis.RDB.Pipeline()
	pipe.Del(ctx, cachekey)
	for _, post := range posts {
		pipe.RPush(ctx, cachekey, post.ID)
	}
	pipe.Expire(ctx, cachekey, 5*time.Minute)
	pipe.Set(ctx, countkey, total, 1*time.Hour)
	pipe.Exec(ctx)
}
