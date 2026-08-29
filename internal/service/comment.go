package service

import (
	"community/internal/db"
	"community/internal/model"
	"errors"

	"gorm.io/gorm"
)

type CommentService struct{}

// CreateComment 发布评论
func (s *CommentService) CreateComment(userID, postID uint, content string) (*model.Comment, error) {
	var post model.Post
	if err := db.DB.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("帖子不存在")
		}
		return nil, err
	}

	comment := model.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
	}
	if err := db.DB.Create(&comment).Error; err != nil {
		return nil, errors.New("发布评论失败")
	}
	return &comment, nil
}

// GetCommentsByPostID 获取帖子的评论列表（手动组装，不依赖 Preload）
func (s *CommentService) GetCommentsByPostID(postID uint) ([]map[string]interface{}, error) {
	// 1. 查评论
	var comments []model.Comment
	if err := db.DB.Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, err
	}

	// 如果没有评论，直接返回空数组
	if len(comments) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 2. 收集所有评论中的 user_id
	userIDs := make([]uint, 0, len(comments))
	for _, c := range comments {
		userIDs = append(userIDs, c.UserID)
	}

	// 3. 一次性查出这些用户
	var users []model.User
	if err := db.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil { //找出所有id在userIDs里面的用户，userIDs里面有重复的id，但是mysql会自动优化去重
		return nil, err
	}

	// 4. 把用户信息转成 map，方便查找
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 5. 组装结果
	result := make([]map[string]interface{}, 0, len(comments))
	for _, c := range comments {
		item := map[string]interface{}{
			"id":         c.ID, //给字典赋值，id是键，c.ID是值
			"post_id":    c.PostID,
			"content":    c.Content,
			"created_at": c.CreatedAt,
		}
		if user, ok := userMap[c.UserID]; ok {
			item["user"] = map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
			}
		} else {
			item["user"] = nil
		}
		result = append(result, item)
	}

	return result, nil
}
