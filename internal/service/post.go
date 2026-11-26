package service

import (
	"community/internal/db"
	"community/internal/model"
	myredis "community/internal/redis"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"

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
func (s *PostService) GetPostList(page, size int, keyword string) ([]model.Post, int64, error) {
	posts := make([]model.Post, 0)
	var total int64

	query := db.DB.Model(&model.Post{})

	if keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where(
			"(title LIKE ? OR content LIKE ?)",
			pattern,
			pattern,
		)
	}

	//计算总数
	if err := query.Count(&total).Error; err != nil { //统计 posts 表里的总记录数，并存到 total 变量里，Model是指定查询的表，Count(&total)执行 SELECT COUNT(*) FROM posts，把结果写入 total
		return nil, 0, err
	}

	//分页查询,按创建时间倒序，预加载用户信息
	offset := (page - 1) * size
	if err := query.
		Preload("User").                   //预加载关联：把每个帖子的 User 信息也查出来，存到 posts[i].User 里
		Order("created_at DESC, id DESC"). //按 created_at 字段降序排列（越新的帖子排越前）
		Offset(offset).                    //跳过前 offset 条数据
		Limit(size).                       //只取 size 条数据
		Find(&posts).Error; err != nil {   //执行查询，结果存入 posts 切片
		return nil, 0, err
	}
	return posts, total, nil
}

// 辅助： GetPostByID 获取帖子详情
func (s *PostService) GetPostByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	if err := db.DB.Preload("User").WithContext(ctx).First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("查询失败：%w", err)
	}
	return &post, nil
}

// 获取帖子详情，并用互斥锁防止缓存击穿
func (s *PostService) GetPostByIDWithLock(ctx context.Context, id uint) (*model.Post, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second) //给函数单独加一个2秒的时间限制，以防用户请求一直不取消，导致无期限抢锁
	defer cancel()
	//先查缓存
	key := fmt.Sprintf("post:%d", id)
	lockKey := key + ":lock"
	var tokenBytes [16]byte                             //创建一个包含 16 个字节的数组,这是装随机数的容器
	if _, err := rand.Read(tokenBytes[:]); err != nil { //把随机字节填进数组。`tokenBytes[:]` 是把整个数组转换成切片，供函数写入
		return nil, fmt.Errorf("生成锁标识失败：%w", err)
	}
	lockToken := hex.EncodeToString(tokenBytes[:]) //把这些字节转换成方便保存和查看的十六进制字符串。一个字节变成两个字符，所以 16 字节会变成 32 个字符
	const unlockScript = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	end
	return 0
	`
	for {
		if err := ctx.Err(); err != nil {
			return nil, ctx.Err()
		}
		val, err := myredis.RDB.Get(ctx, key).Result()
		if err == nil {
			var post model.Post
			if err := json.Unmarshal([]byte(val), &post); err != nil {
				return nil, fmt.Errorf("json解析错误：%w", err)
			}
			return &post, nil
		} else if !errors.Is(err, redis.Nil) { //redis服务不可用，通常get和setnx都会出错，所以不再降级查mysql，Redis出错直接返回
			return nil, fmt.Errorf("Redis服务不可用：%w", err)
		}

		//互斥锁，同一个key只让一个请求查mysql，防止大量请求使数据库崩溃
		ok, err := myredis.RDB.SetNX(ctx, lockKey, lockToken, 10*time.Second).Result() //SetNx的作用是只有key不存在时才设置
		//Result()：执行 Redis 命令，并返回两个值。1.ok：bool 类型。true 表示“这个锁之前不存在，我成功把锁设置进去了，我抢到锁了”；false 表示“锁已经被别人占着，我没抢到”。
		if err != nil {
			return nil, fmt.Errorf("Redis服务不可用：%w", err)
		}
		if ok {
			defer func() {
				unlockCtx, cancel := context.WithTimeout( //释放锁是收尾工作，即使用户的请求已取消，仍需要释放锁
					context.Background(), time.Second,
				)
				defer cancel()

				if err := myredis.RDB.Eval(
					unlockCtx,
					unlockScript,
					[]string{lockKey},
					lockToken,
				).Err(); err != nil {
					log.Printf("释放帖子缓存锁失败，post_id=%d：%v", id, err)
				}
			}()
			//获取锁成功之后再查一次缓存
			val, err := myredis.RDB.Get(ctx, key).Result()
			if err == nil {
				var post model.Post
				if err := json.Unmarshal([]byte(val), &post); err != nil {
					return nil, fmt.Errorf("json解析错误：%w", err)
				}
				return &post, nil
			} else if !errors.Is(err, redis.Nil) { //redis服务不可用，通常get和setnx都会出错，所以不再降级查mysql，Redis出错直接返回
				return nil, fmt.Errorf("Redis服务不可用：%w", err)
			}
			//获取锁成功，查mysql
			post, err := s.GetPostByID(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("mysql查询失败：%w", err)
			}
			//写入缓存
			data, err := json.Marshal(post)
			if err != nil {
				log.Printf("帖子%d序列化失败：%v", id, err)
				return post, nil
			}
			if err := myredis.RDB.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
				log.Printf("帖子%d缓存回填失败：%v", id, err)
			}
			return post, nil
		}

		//获取锁失败，等10ms后循环重试
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond): //等10ms后走到这里
		}
	}
}
