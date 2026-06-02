<h1 align="center">🍀 Get Jobs【工作无忧】</h1>
<div align="center">

## 📖 项目简介

**Get Jobs（工作无忧）** 是一款开源的多平台自动化求职工具，支持 **Boss直聘、猎聘、智联招聘** 四大招聘平台。它能够根据用户配置的搜索条件自动搜索岗位、过滤不合适的职位、并代发打招呼/投递消息，帮助求职者高效批量投递简历。

本项目是基于 [loks666/get_jobs](https://github.com/loks666/get_jobs)（Java/Spring Boot + Playwright 版本）的 **Go 语言重构版**。保留了原项目的核心业务流程与前端界面，后端使用 Go + Gin + go-rod 重写，架构更轻量，部署更简洁。

## 🌟 功能特性

- **🖥️ 图形化界面**：Next.js 网页管理界面，直观配置与运行
- **💥 AI 智能匹配**：支持 DeepSeek / OpenAI 兼容 API，检测岗位匹配度并自动生成打招呼语
- **📷️ 图片简历**：Boss直聘打招呼后自动发送图片简历，提高回复率
- **🔎 多平台支持**：Boss直聘、猎聘、智联招聘
- **🔎 智能过滤**：自动过滤不活跃 HR、猎头岗位、超出薪资范围职位
- **🚫 黑名单**：自动维护公司黑名单，避免重复投递
- **🔄 持久登录**：Cookie 持久化，大部分平台每周仅需扫码一次
- **📢 实时通知**：企业微信消息推送 + SSE 页面实时进度

## 🚀 快速开始

### 前置要求

- Go 1.25+
- Node.js 18+ / Yarn（前端）

### 1️⃣ 克隆并启动后端

```bash
git clone https://github.com/pubaicode/get_jobs_go.git
cd get_jobs_go

go build -o server .
./server
或者

cd front
npm run build:prod
go run main.go
在浏览器中输入 http://localhost:8888

默认只查询不投递，可在配置设置中选择投递模式来开启自动投递
```




后端默认监听 `:8888`，可通过环境变量配置：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SERVER_PORT` | `8888` | 服务端口 |
| `DATABASE_PATH` | `./db/getjobs.db` | SQLite 数据库路径 |
| `FRONTEND_DIR` | `./front` | 前端静态文件目录 |

### 2️⃣ 启动前端

```bash
cd front
yarn install
yarn dev
```

前端默认运行在 `:6866`，自动连接后端 `:8888`。

### 3️⃣ AI 配置

在网页端 **环境变量配置页** 或 `config` 表中设置：

| 配置项 | 说明 |
|---|---|
| `BASE_URL` | API 地址，如 `https://api.deepseek.com` |
| `API_KEY` | API 密钥 |
| `MODEL` | 模型名，如 `deepseek-v4-flash` |

## ⚙️ 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `HOOK_URL` | - | 企业微信机器人 Webhook 地址 |
| `BASE_URL` | - | AI API 基础地址 |
| `API_KEY` | - | AI API 密钥 |
| `MODEL` | - | AI 模型名称 |

## 📁 项目结构

```
├── cmd/server/main.go       # Go 后端入口
├── internal/                 # Go 业务逻辑
│   ├── handler/              # HTTP 处理器
│   ├── service/              # 业务服务层
│   ├── repository/           # 数据访问层
│   ├── model/                # 数据模型
│   ├── worker/               # 平台自动化 Worker
│   ├── manager/              # 浏览器管理
│   ├── middleware/           # 中间件
│   └── database/             # 数据库初始化
├── pkg/sse/                  # SSE 实时推送
├── front/                    # Next.js 前端
└── src/main/                 # 原 Java 版本（参考用）
```

## 📜 开源协议

[MIT](LICENSE)

## 🙏 致谢

- 原版项目：[loks666/get_jobs](https://github.com/loks666/get_jobs) — Java/Spring Boot 版本
- 所有贡献者和使用者
