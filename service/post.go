package service

import (
	"community/db"
	"community/model"
	"community/redis"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type PostService struct{}

// CreatePost 发布帖子
func (s *PostService) CreatePost(userID uint, title, content string) (*model.Post, error) {
	post := model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	if err := db.DB.Create(&post).Error; err != nil {
		return nil, errors.New("发布帖子失败")
	}
	return &post, nil
}

// GetPostList 获取帖子列表（分页）
func (s *PostService) GetPostList(page, size int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	//计算总数
	if err := db.DB.Model(&model.Post{}).Count(&total).Error; err != nil { //统计 posts 表里的总记录数，并存到 total 变量里，Model是指定查询的表，Count(&total)执行 SELECT COUNT(*) FROM posts，把结果写入 total
		return nil, 0, nil
	}

	//分页查询,按创建时间倒序，预加载用户信息
	offset := (page - 1) * size
	if err := db.DB.
		Preload("User").                 //预加载关联：把每个帖子的 User 信息也查出来，存到 posts[i].User 里
		Order("created_at DESC").        //按 created_at 字段降序排列（越新的帖子排越前）
		Offset(offset).                  //跳过前 offset 条数据
		Limit(size).                     //只取 size 条数据
		Find(&posts).Error; err != nil { //执行查询，结果存入 posts 切片
		return nil, 0, err
	}
	return posts, total, nil
}

// 辅助： GetPostByID 获取帖子详情
func (s *PostService) GetPostByID(id uint) (*model.Post, error) {
	var post model.Post
	if err := db.DB.Preload("User").First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("帖子不存在")
		}
		return nil, err
	}
	return &post, nil
}

// 获取帖子详情，并用互斥锁防止缓存击穿
func (s *PostService) GetPostByIDWithLock(ctx context.Context, id uint) (*model.Post, error) {
	//先查缓存
	key := fmt.Sprintf("post:%d", id)
	val, err := redis.RDB.Get(ctx, key).Result()
	if err == nil {
		var post model.Post
		json.Unmarshal([]byte(val), &post)
		return &post, nil
	}

	//互斥锁，同一个key只让一个请求查mysql，防止大量请求使数据库崩溃
	lockKey := key + ":lock"
	ok, err := redis.RDB.SetNX(ctx, lockKey, "1", 10*time.Second).Result() //"1"：锁的值，可以简单理解为“占用标志”，只要能设置进去，就意味着抢锁成功
	//Result()：执行 Redis 命令，并返回两个值。1.ok：bool 类型。true 表示“这个锁之前不存在，我成功把锁设置进去了，我抢到锁了”；false 表示“锁已经被别人占着，我没抢到”。
	if ok {
		//获取锁成功，查mysql
		post, err := s.GetPostByID(id)
		if err != nil {
			return nil, err
		}
		//写入缓存
		data, _ := json.Marshal(post)
		redis.RDB.Set(ctx, key, data, 5*time.Minute)
		redis.RDB.Del(ctx, lockKey) //释放锁
		return post, nil
	}

	//获取锁失败，等10ms后递归重试
	time.Sleep(10 * time.Millisecond)
	return s.GetPostByIDWithLock(ctx, id)
}
