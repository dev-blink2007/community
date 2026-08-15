# CommunityHub - 社区后端服务系统
> 一个基于 Go + Gin + GORM + MySQL + Redis 的内容社区后端系统，支持用户登录（使用 bcrypt 生成加盐加密密码）和认证（JWT，token 有效期 24 小时），帖子管理、评论互动、点赞、关注和 Feed 流，集成了 Redis 缓存优化（互斥锁防缓存击穿），支持热榜排序并定时刷新。

## 项目亮点
- **高并发点赞** : 使用 Redis 实现点赞计数，并定时同步到 MySQL
- **Feed流** : 获取关注用户所有帖子的动态列表并按时间降序排列，支持分页和缓存
- **热榜排序** : 基于Redis ZSET这个集合实现帖子热度前100的动态排序
- **缓存优化** : 旁路缓存模式 + 互斥锁防缓存击穿（适用于读多写少的场景）
- **分层架构** : 使用三层架构（model/service/controller）

## 技术栈
|   技术   |   用途   |
|----------|----------|
| Go 1.23  | 后端语言  |
|   Gin    | web框架  |
|   GORM   | ORM 框架 |
| MySQL 8.0| 主数据库 |
| Redis 7.0| 缓存+热榜|

## 项目结构
community/
├── config/config.go           # 配置加载（环境变量）
├── controller/                # 控制器层（处理 HTTP 请求）
│   ├── user.go
│   ├── post.go
│   ├── comment.go
│   ├── like.go
│   ├── follow.go
│   ├── feed.go
│   └── hot.go
├── db/
│   ├── db.go               # MySQL 连接与初始化 + GORM自动建表
├── middleware/
│   └── auth.go                # JWT 认证中间件
├── model/                     # 数据模型定义
├── redis/
│   └── redis.go               # Redis 连接与初始化
├── service/                   # 业务逻辑层
├── utils/                     # 工具函数
├── bench/
│   └── bench.go               # 自研并发压测脚本
├── main.go                    # 入口 + 路由注册 + 定时任务
└── go.mod

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

服务器启动在 `http://localhost:8080`

## API接口
|  方法  |             路径             |    描述    |  认证  |
|--------|-----------------------------|------------|--------|
|  POST  |        /api/register        |   用户注册  |   否   |
|  POST  |         /api/login          |   用户登录  |   否   |
|  GET   |         /api/posts          | 获取帖子列表|   否   |
|  GET   |       /api/post/:id         |   获取帖子  |   否   |
|  GET   |/api/posts/:post_id/comments |   获取评论  |   否   |
|  GET   |/api/posts/:post_id/like-status|查询是否点赞|   否   |
|  GET   |/api/posts/:post_id/like-count| 获取点赞数  |   否   |
|  GET   |      /api/posts/hot         |  获取热榜帖子|   否   |
|  POST  |         /api/posts          |   发布帖子   |   是   |
|  POST  |        /api/comments        |   发布评论   |   是   |
|  POST  |/api/posts/:post_id/like     | 点赞/取消点赞|   是   |
|  POST  |/api/users/:user_id/follow   | 关注/取消关注|   是   |
|  GET   |/api/users/:user_id/followings|  获取关注列表|   是   |
|  GET   |/api/users/:user_id/followers|  获取粉丝列表 |   是   |
|  GET   |         /api/feed           |  获取Feed流  |   是   |
|  GET   |   /api/admin/refresh-hot    |    刷新热榜   |   是   |

## 性能测试
- **测试环境** ：Windows 11 + Go 1.23 + MySQL 8.0
- **测试工具** ：Go 自研并发压测脚本
- **测试方法** ：50并发 × 1000请求

| 接口 | QPS | 平均延迟 | 说明 |
|------|-----|----------|------|
| `GET /api/posts` | 271 | 3.7ms | 列表查询（带缓存），代表读接口性能基准 |

> 压测脚本位于 `bench/bench.go`，可扩展测试更多接口

## 线上地址
http://8.137.190.152/api/posts
