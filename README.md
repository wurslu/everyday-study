# 📚 每日学习助手 API

> 基于 AI 的智能学习内容推荐系统，每天为你推送精选的英语谚语、中文古诗词和中医养生知识。

## ✨ 特性

- 🤖 **AI 智能推荐**: 集成豆包大模型，生成高质量学习内容
- 🔄 **防重复机制**: 智能避免推荐已学过的内容
- 📅 **每日缓存**: 同一天所有用户看到相同的精选内容
- 🌐 **全球共享**: 无需注册，所有用户共享学习资源
- 📊 **学习统计**: 提供详细的学习历史和统计数据
- 🚀 **高性能**: Go 语言开发，响应迅速

## 📖 学习内容类型

### 🇺🇸 英语谚语 (`english`)

- 精选有教育意义的英语传统谚语、格言、习语
- 提供中文释义和关键词汇解析
- 帮助提升英语理解能力

### 🇨🇳 中文古诗词 (`chinese`)

- 经典古诗、词、赋等传统文化瑰宝
- 包含文化背景和诗词解释
- 传承中华传统文化精髓

### 🌿 中医养生 (`tcm`)

- 《黄帝内经》、《伤寒论》等经典条文
- 实用的中医理论和养生方法
- 提供临床意义和应用指导

## 🚀 快速开始

### 在线体验

API 已部署到 Vercel，可直接访问：

```bash
# 健康检查
curl https://your-app.vercel.app/api/health

# 获取今日英语学习内容
curl https://your-app.vercel.app/api/today-learning/english

# 获取学习历史
curl https://your-app.vercel.app/api/learning-history
```

### 本地开发

1. **克隆项目**

```bash
git clone https://github.com/your-username/everyday-study-backend.git
cd everyday-study-backend
```

2. **安装依赖**

```bash
go mod tidy
```

3. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，添加你的 API 密钥
```

4. **运行项目**

```bash
go run main.go
```

服务将在 `http://localhost:91` 启动。

## 📡 API 接口

### 基础 URL

```
https://your-app.vercel.app/api
```

### 接口列表

#### 1. 健康检查

```http
GET /api/health
```

**响应示例:**

```json
{
	"success": true,
	"message": "服务运行正常",
	"data": {
		"status": "ok",
		"database": "connected",
		"supported_types": ["english", "chinese", "tcm"]
	}
}
```

#### 2. 获取今日学习内容

```http
GET /api/today-learning/{type}
```

**参数:**

- `type`: 学习类型 (`english` | `chinese` | `tcm`)

**响应示例:**

```json
{
	"success": true,
	"message": "获取今日学习内容成功",
	"data": {
		"type": "english",
		"type_name": "英语",
		"content": "Actions speak louder than words",
		"interpretation": "行动胜过言语。意思是实际行动比空洞的话语更有说服力...",
		"key_words": ["actions: 行动", "speak: 说话", "louder: 更响亮的"],
		"date": "2024-12-24",
		"from_cache": true
	}
}
```

#### 3. 获取学习历史

```http
GET /api/learning-history[?limit=10]
GET /api/learning-history/{type}[?limit=10]
```

**查询参数:**

- `limit`: 返回记录数量，默认 10

#### 4. 获取学习统计

```http
GET /api/stats
```

**响应示例:**

```json
{
	"success": true,
	"message": "获取统计信息成功",
	"data": {
		"stats": {
			"english": {
				"type_name": "英语",
				"total_days": 15,
				"unique_days": 12
			}
		}
	}
}
```

## 🔧 技术栈

- **后端框架**: [Gin](https://gin-gonic.com/) - 高性能 Go Web 框架
- **数据库**: [SQLite](https://sqlite.org/) + [GORM](https://gorm.io/) - 轻量级数据库
- **AI 服务**: [豆包大模型](https://www.volcengine.com/product/doubao) - 字节跳动大模型服务
- **部署平台**: [Vercel](https://vercel.com/) - 无服务器部署

## 📁 项目结构

```
everyday-study-backend/
├── main.go                     # 应用入口
├── go.mod                      # Go 模块依赖
├── .env.example               # 环境变量模板
├── README.md                  # 项目说明
├── internal/                  # 内部包
│   ├── config/               # 配置管理
│   │   └── config.go
│   ├── models/               # 数据模型
│   │   └── models.go
│   ├── database/             # 数据库操作
│   │   └── database.go
│   ├── api/                  # 外部 API 调用
│   │   └── volcano.go
│   ├── middleware/           # 中间件
│   │   └── error.go
│   └── handlers/             # HTTP 处理器
│       └── handlers.go
└── vercel.json               # Vercel 部署配置
```

## 🌍 部署到 Vercel

### 一键部署

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/your-username/everyday-study-backend)

### 手动部署

1. **Fork 本项目**

2. **在 Vercel 中导入项目**

   - 访问 [Vercel Dashboard](https://vercel.com/dashboard)
   - 点击 "New Project"
   - 导入你的 GitHub 仓库

3. **配置环境变量**

   ```
   ARK_API_KEY=你的豆包API密钥
   ENVIRONMENT=production
   ```

4. **部署完成**
   - Vercel 会自动构建和部署
   - 获取你的专属 API 地址

### Vercel 配置文件

项目包含 `vercel.json` 配置文件，支持：

- Go 函数自动路由
- CORS 跨域支持
- 环境变量配置
- 自动 HTTPS

## 📋 环境变量

创建 `.env` 文件并配置以下变量：

```bash
# 服务器配置
PORT=91
ENVIRONMENT=development

# 数据库配置
DATABASE_PATH=learning.db

# AI API 配置
ARK_API_KEY=your_ark_api_key_here
VOLCANO_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
```

## 🔨 开发指南

### 本地开发

```bash
# 安装 Air 热重载工具
go install github.com/cosmtrek/air@latest

# 启动热重载开发
air
```

### 代码规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方编码规范
- 提交前运行测试

### 测试

```bash
# 运行测试
go test ./...

# 测试覆盖率
go test -cover ./...
```

## 📊 使用示例

### JavaScript/Node.js

```javascript
const API_BASE = "https://your-app.vercel.app/api";

// 获取今日英语学习内容
async function getTodayEnglish() {
	const response = await fetch(`${API_BASE}/today-learning/english`);
	const data = await response.json();
	console.log(data.data.content);
}

// 获取学习历史
async function getHistory() {
	const response = await fetch(`${API_BASE}/learning-history?limit=5`);
	const data = await response.json();
	return data.data.records;
}
```

### Python

```python
import requests

API_BASE = "https://your-app.vercel.app/api"

# 获取今日中医学习内容
def get_today_tcm():
    response = requests.get(f"{API_BASE}/today-learning/tcm")
    data = response.json()
    return data["data"]

# 获取统计信息
def get_stats():
    response = requests.get(f"{API_BASE}/stats")
    return response.json()["data"]["stats"]
```

### curl 命令

```bash
# 获取今日中文古诗词
curl -X GET "https://your-app.vercel.app/api/today-learning/chinese" \
  -H "Accept: application/json"

# 获取最近5条学习记录
curl -X GET "https://your-app.vercel.app/api/learning-history?limit=5" \
  -H "Accept: application/json"
```

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 📝 更新日志

### v1.0.0 (2024-12-24)

- ✨ 初始版本发布
- 🚀 支持三种学习内容类型
- 🤖 集成豆包大模型 API
- 📦 Vercel 无服务器部署支持

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源协议。

## 💡 特别感谢

- [Gin Web Framework](https://gin-gonic.com/) - 优秀的 Go Web 框架
- [GORM](https://gorm.io/) - 强大的 Go ORM 库
- [豆包大模型](https://www.volcengine.com/product/doubao) - 提供 AI 能力支持
- [Vercel](https://vercel.com/) - 优秀的部署平台

## 📞 联系方式

- 项目地址: [GitHub](https://github.com/your-username/everyday-study-backend)
- 问题反馈: [Issues](https://github.com/your-username/everyday-study-backend/issues)
- 作者邮箱: your-email@example.com

---

⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！
