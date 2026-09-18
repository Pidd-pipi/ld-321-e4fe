# AgriDispatch（农机调度管理系统）

> 项目类型：全栈Web应用

面向农业合作社与种植大户的农机作业调度平台，支持农机资源管理、作业任务指派（推荐空闲农机/驾驶员）、实时地图轨迹监控（WebSocket + 高德地图）、作业记录统计报表、维修保养提醒与驾驶员管理。

## 快速启动（Docker Compose 一键部署，首选）

```bash
# 1. 首次启动前复制环境变量
cp .env.example .env

# 2. 启动全部服务（MySQL + Redis + 后端 + 前端）
docker compose up -d

# 3. 查看健康状态
docker compose ps
```

访问地址：

- 前端调度看板：http://localhost:18621
- 后端 API：http://localhost:19621
- 健康检查：http://localhost:19621/healthz
- Swagger：http://localhost:19621/docs/index.html

演示账号：`admin / admin123`

## 派单预占闭环

提交派单时只有同时满足以下三个条件，系统才会在**单个数据库事务**内把任务、农机、驾驶员三方一起标记为作业中并写入绑定关系：

1. 农机当前为「空闲」（未被其他任务占用、非维修中）；
2. 农机「保养剩余工时」不少于任务「预计作业时长」；
3. 驾驶员当天「在岗」且未被其他任务占用。

任一条件不满足则**整单拒绝**（HTTP 409，返回明确中文原因），事务回滚，不会出现只占用农机或只占用驾驶员的单向残留；每次尝试（成功/拒绝/释放/重复取消）都写入 `dispatch_attempts` 并回写任务最近尝试字段，看板刷新后仍可回读失败原因。

- **取消派单**：`POST /api/v1/tasks/:id/cancel` 在单事务内同步释放任务、农机（状态回空闲、解绑任务）、驾驶员（占用态回空闲、解绑任务）。
- **并发/重复**：进程内互斥 + 事务内固定顺序（任务→农机→驾驶员）`SELECT ... FOR UPDATE` 行锁 + 条件式 `UPDATE ... WHERE status=...` 三重保障，重复或并发的派单/取消只有一次真正生效。
- 看板的「作业任务调度」展示三方绑定关系、最近结果与失败原因，并提供取消按钮；「派单记录」面板展示全部尝试留痕。

## 本地开发

```bash
# 前端（Vue 3 + TS + Vite + Element Plus + ECharts）
cd frontend
npm install
npm run dev        # http://localhost:18621

# 后端（Go）
cd backend
go mod tidy
go run ./cmd/server
```

## 技术栈

| 层次 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus + ECharts |
| 地图 | 高德地图 API（JSAPI） |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7（go-redis/v9，看板缓存 + 实时推送） |
| 实时通信 | WebSocket（gorilla/websocket，农机实时位置推送） |
| 认证 | JWT (github.com/golang-jwt/jwt/v5) + RBAC |
| 配置 | github.com/caarlos0/env/v11 |
| 日志 | Go 标准库 log/slog |

## 项目目录结构

```
.
├── docker-compose.yml          # db + redis + backend + frontend
├── .env / .env.example
├── README.md
├── frontend/
│   ├── Dockerfile / nginx.conf # /api/ → backend:8080/api/v1/，/ws → backend:8080/ws
│   └── src/
│       ├── features/           # DashboardView
│       ├── components/         # MachineTable/TaskBoard/MapTrackPanel/...
│       ├── services/           # API 调用（overview/dispatch）
│       ├── stores/  types/  constants/  logger/  errors/
└── backend/
    ├── Dockerfile
    ├── database/init.sql
    ├── cmd/server/main.go
    └── internal/
        ├── config/  model/  repository/  service/  handler/
        ├── router/  middleware/  ws/          # WebSocket Hub
        ├── constants/  errors/  logger/  util/  database/
```

## 主要 API 列表

统一前缀 `/api/v1`，统一响应 `{ "code": 0, "message": "ok", "data": ... }`。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | /auth/login | 登录 | - |
| GET | /auth/me | 当前用户 | JWT |
| GET | /dashboard/overview | 调度看板总览（农机/任务/轨迹/统计/保养/驾驶员/派单记录） | - |
| POST | /tasks/:id/dispatch | 派单预占：农机空闲 + 保养剩余工时≥预计时长 + 驾驶员当天在岗，三者同时满足才原子占用，否则整单拒绝（409，返回失败原因） | - |
| POST | /tasks/:id/cancel | 取消派单：单事务同步释放任务/农机/驾驶员；重复或并发取消仅首次真正生效 | - |
| GET | /dashboard/reports/work/export | 作业报表导出信息 | - |
| GET | /ws | WebSocket 实时轨迹推送 | - |
| GET | /healthz | 健康检查（DB + Redis） | - |

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | Compose 项目名 | agridispatch |
| APP_ENV | 运行环境 | development |
| SERVER_PORT | 后端端口（容器内） | 8080 |
| DB_HOST / DB_PORT | 数据库地址/端口 | db / 3306 |
| DB_NAME / DB_USER / DB_PASSWORD | 数据库配置 | agridispatch |
| DB_ROOT_PASSWORD | MySQL root 密码 | agridispatch_root_pwd |
| REDIS_HOST / REDIS_PORT | Redis 地址/端口 | redis / 6379 |
| REDIS_PASSWORD | Redis 密码 | 空 |
| JWT_SECRET | JWT 密钥 | 请修改 |
| FRONTEND_PORT | 前端宿主端口 | 18621 |
| BACKEND_PORT | 后端宿主端口 | 19621 |
| DB_PORT | MySQL 宿主端口 | 33321 |
| REDIS_PORT | Redis 宿主端口 | 36321 |

## Docker 部署说明

- 端口映射：前端 `18621:80`、后端 `19621:8080`、MySQL `33321:3306`、Redis `36321:6379`
- 数据卷：`db_data`、`redis_data`
- Nginx 反代：`/api/ → http://backend:8080/api/v1/`；`/ws` 配置 WebSocket Upgrade/Connection 头
- 常见问题：等待 `docker compose ps` 全部 healthy（MySQL 初始化约 30~120 秒）；高德地图 Key 在线上环境需按 `MapTrackPanel` 接入真实 JSAPI Key

## License

MIT License
