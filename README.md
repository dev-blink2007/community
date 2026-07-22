# CommunityHub - 社区后端服务系统
> 一个基于Go + Gin + GORM + MySQL + Redis 的内容社区后端系统，支持用户认证，帖子管理，评论互动，点赞，关注和Feed流，并集成了Redis缓存优化

## 项目亮点
- **高并发点赞** : 使用Redis实现点赞技术，并定时同步到mysql
- **Feed流** : 获取关注用户所有帖子的动态列表并按时间降序排列，支持分页和缓存
- **热榜排序** : 基于Redis ZSET这个集合实现帖子热度前100的动态排序
- **缓存优化** : 旁路缓存模式 + 互斥锁防缓存击穿（使用读多写少的场景）
- **分层架构** : 使用三层架构（model/service/controller）

## 技术栈
|   技术   |   用途   |
|----------|----------|
| Go 1.26  | 后端语言  |
|   Gin    | web框架  |
|   GORM   | ORM 框架 |
| MySQL 8.0| 主数据库 |
| Redis 7.0| 缓存+热榜|

## 项目结构
community/
|———— config/# 配置管理
|———— service/# 业务逻辑
|———— model/# 数据模型
|———— controller/# 控制器层（请求管理）
|———— db /# 数据库连接
|———— redis /# Redis连接
|———— utils /# 工具函数
|———— middleware /# 中间件（jwt，加密密码）
|———— .env /# 环境变量配置
|———— main.go /#程序入口

## 快速启动
### 1.克隆项目
\`\`\`bash
git clone https://github.com/dev-blink2007/community.git
cd community
\`\`\`

### 2.安装依赖
\`\`\`bash
go mod download
\`\`\`

### 3.配置环境变量
复制`.env.example` 为 `.env` ,修改数据库和Redis配置

### 4.创建数据库
\`\`\`sql
CREATE DATABASE community;
\`\`\`

### 5.运行项目
\`\`\`bash
go run main.go
\`\`\`

服务器启动在`http://localhost//:8080`

## API接口
|  方法  |             路径             |    描述    |  认证  |
|--------|-----------------------------|------------|--------|
|  POST  |        /api/register        |   用户注册  |   否   |
|  POST  |         /api/login          |   用户登陆  |   否   |
|  GET   |         /api/posts          | 获取帖子列表|   否   |
|  GET   |       /api/post/:id         |   获取帖子  |   否   |
|  GET   |/api/posts/:post_id/comments |   获取评论  |   否   |
|  GET   |/api/posts/:post_id/like-status|查询是否点赞|   否   |
|  GET   |/api/posts/:post_id/like-count| 获取点赞数  |   否   |
|  GET   |      /api/posts/hot         |  获取热榜帖子|   否   |
|  POST  |         /api/posts          |   发布帖子   |   是   |
|  POST  |        /api/comments        |   发布评论   |   是   |
|  POST  |/api/posts/:post_id/like     | 点赞/取消点赞|   是   | 
|  GET   |/api/users/:user_id/followings|  获取关注列表|  是   |
|  GET   |/api/users/:user_id/followers|  获取粉丝列表 |  是   |
|  GET   |         /api/feed           |  获取Feed流  |   是   |
|  GET   |   /api/admin/refresh-hot    |    刷新热榜   |   是   |

## 性能测试
- **测试环境** ：Windows 11 + Go 1.26 + Mysql 8.0
- **测试工具** ：Go自研并发压测脚本
- **测试方法** ：50并发*1000请求

| 接口 | QPS | 平均延迟 | 说明 |
|------|-----|----------|------|
| `GET /api/posts` | 271 | 3.7ms | 列表查询，代表读接口性能基准 |

> 其他接口（如详情、点赞）暂未压测，预期性能与此相近或略低
