package service

import (
	"community/db"
	"community/model"
	myredis "community/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type LikeService struct{}

// ToggleLike 点赞/取消点赞
// 返回值（是否已点赞，error）
func (s *LikeService) ToggleLike(ctx context.Context, userID, postID uint) (bool, error) {
	var like model.Like
	key := fmt.Sprintf("like:count:%d", postID)

	//查询是否已经点赞
	err := db.DB.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error //这里用到了联合索引，因为 WHERE 条件正好是 user_id 和 post_id，且顺序和索引定义一致，数据库会直接走索引查找，复杂度从 O(n) 降到 O(log n)
	if err == nil {
		//已点赞--取消点赞
		if err := db.DB.Delete(&like).Error; err != nil { //删除之后主键id不会回填
			return false, err
		}
		if err := myredis.RDB.Decr(ctx, key).Err(); err != nil {
			//redis操作失败只记录日志不影响返回，因为mysql可以确保数据一致性
			log.Printf("redis Decr 失败:%v", err)
		}
		return false, nil //这里的false是指操作之后点赞的状态
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		//未点赞，创建点赞记录
		like = model.Like{
			UserID: userID,
			PostID: postID,
		}
		if err := db.DB.Create(&like).Error; err != nil {
			return false, err
		}
		if err := myredis.RDB.Incr(ctx, key).Err(); err != nil {
			log.Printf("redis Incr 失败:%v", err)
		}
		return true, nil
	} else {
		return false, err
	}

}

// GetLikeStatus 查询用户是否已点赞
func (s *LikeService) GetLikeStatus(userID, postID uint) (bool, error) {
	var like model.Like
	err := db.DB.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error
	if err == nil {
		return true, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

// 从Redis获取点赞数
func (s *LikeService) GetLikeCountWithCache(ctx context.Context, postID uint) (int64, error) {
	key := fmt.Sprintf("like:count:%d", postID)

	//先查Redis
	val, err := myredis.RDB.Get(ctx, key).Int64()
	if err == nil {
		return val, nil
	}

	//Redis没命中，再查mysql
	var count int64
	if err := db.DB.Model(&model.Like{}).Where("post_id = ?", postID).Count(&count).Error; err != nil {
		return 0, err
	}

	//写入Redis
	myredis.RDB.Set(ctx, key, count, 1*time.Hour)
	return count, nil
}

// 将Redis点赞数定时同步到mysql
func (s *LikeService) SyncLikeCountToMySQL(ctx context.Context) error {
	var cursor uint64 //默认值是0
	var keys []string

	//用scan遍历所有like:count:*的key
	for {
		var batch []string
		var err error
		batch, cursor, err = myredis.RDB.Scan(ctx, cursor, "like:count:*", 100).Result() //cursor是开始扫描的下标，每次扫描100是出于性能考虑
		if err != nil {
			return err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
		if len(keys) == 0 {
			log.Println("没有需要同步的点赞数")
			return nil
		}

		log.Printf("找到%d个需要同步的点赞数", len(keys))

		//批量获取Redis里面的值
		pipe := myredis.RDB.Pipeline()
		cmds := make([]*redis.StringCmd, len(keys)) //redis.StringCmd:一个结构体，里面存了 GET 命令的返回值和错误信息;*redis.StringCmd:指向这个结构体的指针，避免复制数据
		for i, key := range keys {
			cmds[i] = myredis.RDB.Get(ctx, key)
		}
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) { //redis.Nil是key不存在的一个特殊错误，不存在的key跳过就行了
			return err
		}

		//解析keys里面的post_id
		for i, key := range keys {
			var postID uint
			fmt.Sscanf(key, "like:count:%d", postID)
			val, err := cmds[i].Int64()
			if err != nil {
				log.Printf("获取key %s 失败：%v", key, err)
				continue
			}

			//更新mysql
			if err := db.DB.Model(&model.Post{}).Where("id = ?", postID).Update("like_count", val).Error; err != nil {
				log.Printf("更新帖子 %d 失败: %v", postID, err)
				continue
			}
			log.Printf("同步成功：帖子 %d 点赞数 %d", postID, val)
		}
	}
	return nil
}
