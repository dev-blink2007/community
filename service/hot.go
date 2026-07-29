package service

import (
	"community/db"
	"community/model"
	myredis "community/redis"
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
)

type HotService struct{}

const hotKey = "hot:posts"

// 计算单篇帖子的热度,只是一个单纯的计算工具，不用绑定成方法
func calcHotScore(post model.Post, likeCount, commentCount int64) float64 {
	//基础分 = 点赞*3 + 评论*3
	baseScore := float64(likeCount*3 + commentCount*2)

	//时间衰减：每过一小时减一分
	hours := time.Since(post.CreatedAt).Hours()
	dacay := math.Max(0, hours/1.0)

	return baseScore - dacay
}

// RefreshHot 刷新热榜（定时任务调用）
func (s *HotService) RefreshHot(ctx context.Context) error { //context用来管理请求生命周期的容器
	var posts []model.Post
	//只查最近7天的帖子
	weekAgo := time.Now().AddDate(0, 0, -7)
	if err := db.DB.Where("created_at > ?", weekAgo).Find(&posts).Error; err != nil {
		return err
	}
	log.Printf("查到 %d 篇帖子", len(posts))
	pipe := myredis.RDB.Pipeline() //将代码打包进管道，管道里的代码统一传输，减少I/O
	pipe.Del(ctx, hotKey)
	for _, post := range posts {
		//获取点赞数
		var likeCount int64
		if err := db.DB.Model(&model.Like{}).Where("post_id = ?", post.ID).Count(&likeCount).Error; err != nil {
			return err
		}

		//获取评论数
		var commentCount int64
		if err := db.DB.Model(&model.Comment{}).Where("post_id = ? ", post.ID).Count(&commentCount).Error; err != nil {
			return err
		}

		//计算热度
		score := calcHotScore(post, likeCount, commentCount)

		//写入Redis ZSET(一个有序集合，每个元素都挺对应一个分数)
		pipe.ZAdd(ctx, hotKey, &redis.Z{
			Score:  score,
			Member: post.ID,
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

// 获取热榜帖子
func (s *HotService) GetHotPost(ctx context.Context, limit int) ([]model.Post, error) {
	//从Redis取top n帖子的ID
	ids, err := myredis.RDB.ZRevRange(ctx, hotKey, 0, int64(limit-1)).Result() //ids[]返回的是字符串切片
	//ZRevRange按得分大小排序
	// 0:起始索引（从第 0 个开始，即分数最高的）,int64(limit-1):结束索引（如果 limit=10，结束索引是 9，取前 10 个）,.Result():执行命令并返回结果
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []model.Post{}, nil
	}

	//转成uint切片
	var postIDs []uint
	for _, idstr := range ids {
		var id uint
		fmt.Sscanf(idstr, "%d", &id) //对字符串进行操作将其变换成指定格式，%d是指定变成整数类型
		postIDs = append(postIDs, id)
	}

	//从数据库查帖子详情(按ID顺序升序，而不是传入的热度顺序)
	var posts []model.Post
	if err := db.DB.Where("id IN ?", postIDs).Preload("User").Find(&posts).Error; err != nil {
		return nil, err
	}

	//按redis返回的顺序重新排列
	postMap := make(map[uint]model.Post)
	for _, p := range posts {
		postMap[p.ID] = p
	}
	orderd := make([]model.Post, 0, len(postIDs))
	for _, id := range postIDs {
		if p, ok := postMap[id]; ok {
			orderd = append(orderd, p)
		}
	}
	return orderd, nil
}
