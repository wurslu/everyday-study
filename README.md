# 📚 每日学习助手 API

> 基于豆包大模型的智能学习内容推荐系统，每天为全球用户推送精选的英语谚语、中文古诗词和中医养生知识。

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![API Status](https://img.shields.io/badge/API-Online-brightgreen.svg)](https://everyday-study-backend.onrender.com/api/health)
[![Deploy on Render](https://img.shields.io/badge/Deploy-Render-46E3B7.svg)](https://render.com)

## 🌟 在线体验

**🚀 API 基础地址**: https://everyday-study-backend.onrender.com

**🔍 快速测试**:

- [健康检查](https://everyday-study-backend.onrender.com/api/health)
- [今日英语学习](https://everyday-study-backend.onrender.com/api/today-learning/english)
- [今日中文学习](https://everyday-study-backend.onrender.com/api/today-learning/chinese)
- [今日中医学习](https://everyday-study-backend.onrender.com/api/today-learning/tcm)

## ✨ 项目特性

- 🤖 **AI 智能推荐**: 集成豆包大模型，生成高质量学习内容
- 🔄 **防重复机制**: 智能避免推荐已学过的内容
- 📅 **全球共享**: 同一天所有用户看到相同的精选内容
- 🌐 **无需注册**: 开箱即用，无需用户管理
- 📊 **学习统计**: 提供详细的学习历史和统计数据
- 🚀 **高性能**: Go 语言开发，响应迅速
- 🐳 **容器化**: 支持 Docker 一键部署
- ☁️ **云端部署**: 已部署到 Render 云平台

## 📖 学习内容类型

### 🇺🇸 英语谚语 (`english`)

- **内容来源**: 英语传统谚语、格言、习语
- **返回格式**: 谚语原文 + 中文释义 + 关键词解析
- **学习价值**: 提升英语理解能力和文化素养

**示例响应**:

```json
{
	"success": true,
	"data": {
		"type": "english",
		"content": "Actions speak louder than words",
		"interpretation": "行动胜过言语。意思是实际行动比空洞的话语更有说服力...",
		"key_words": ["actions: 行动", "speak: 说话", "louder: 更响亮的"],
		"from_cache": true
	}
}
```

### 🇨🇳 中文古诗词 (`chinese`)

- **内容来源**: 古诗、词、赋等传统文化瑰宝
- **返回格式**: 诗词原文 + 文化背景 + 意境解析
- **学习价值**: 传承中华传统文化精髓

### 🌿 中医养生 (`tcm`)

- **内容来源**: 《黄帝内经》、《伤寒论》等经典条文
- **返回格式**: 条文原文 + 临床意义 + 应用指导
- **学习价值**: 了解中医理论和养生方法

## 🚀 快速开始

### 在线使用（推荐）

无需安装，直接调用在线 API：

```bash
# 获取今日英语学习内容
curl https://everyday-study-backend.onrender.com/api/today-learning/english

# 获取学习历史
curl https://everyday-study-backend.onrender.com/api/learning-history

# 获取统计信息
curl https://everyday-study-backend.onrender.com/api/stats
```

### 本地开发

1. **克隆项目**

```bash
git clone https://github.com/wurslu/everyday-study-backend.git
cd everyday-study-backend
```

2. **安装依赖**

```bash
go mod tidy
```

3. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，添加你的豆包 API 密钥
```

4. **运行项目**

```bash
go run main.go
```

服务将在 `http://localhost:91` 启动。

## 📡 API 接口文档

### 基础信息

- **基础 URL**: `https://everyday-study-backend.onrender.com/api`
- **请求方式**: GET
- **响应格式**: JSON
- **编码**: UTF-8

### 接口列表

#### 1. 健康检查

```http
GET /api/health
```

**响应示例**:

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

**参数说明**:

- `type`: 学习类型
  - `english` - 英语谚语
  - `chinese` - 中文古诗词
  - `tcm` - 中医养生

**特点**:

- ✅ 同一天返回相同内容（全局缓存）
- ✅ 防重复推荐机制
- ✅ AI 智能生成

#### 3. 获取学习历史

```http
GET /api/learning-history[?limit=10]
GET /api/learning-history/{type}[?limit=10]
```

**查询参数**:

- `limit`: 返回记录数量（默认 10，最大 100）

#### 4. 获取学习统计

```http
GET /api/stats
```

**响应包含**:

- 各类型学习总天数
- 不重复学习天数
- 学习类型分布

## 🔧 技术架构

### 后端技术栈

- **框架**: [Gin](https://gin-gonic.com/) - 高性能 Go Web 框架
- **数据库**: [SQLite](https://sqlite.org/) + [GORM](https://gorm.io/) - 轻量级 ORM
- **AI 服务**: [豆包大模型](https://www.volcengine.com/product/doubao) - 字节跳动大模型
- **部署**: [Render](https://render.com/) - 云端部署平台

### 项目结构

```
everyday-study-backend/
├── main.go                    # 应用入口
├── go.mod                     # Go 模块依赖
├── .env.example              # 环境变量模板
├── Dockerfile                # Docker 构建文件
├── docker-compose.yml        # Docker 编排文件
├── deploy.sh                 # 一键部署脚本
├── README.md                 # 项目文档
├── internal/                 # 内部包
│   ├── config/              # 配置管理
│   ├── models/              # 数据模型
│   ├── database/            # 数据库操作
│   ├── api/                 # 外部 API 调用
│   ├── middleware/          # 中间件
│   └── handlers/            # HTTP 处理器
└── .github/                 # GitHub 工作流（可选）
```

## 🐳 Docker 部署

### 快速部署

1. **克隆项目**

```bash
git clone https://github.com/wurslu/everyday-study-backend.git
cd everyday-study-backend
```

2. **配置环境变量**

```bash
cp .env.docker .env
# 编辑 .env 文件，填入真实的 ARK_API_KEY
```

3. **一键部署**

```bash
chmod +x deploy.sh
./deploy.sh
```

### 手动部署

```bash
# 构建并启动
docker-compose up -d --build

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## ☁️ 云端部署

### Render 部署（当前使用）

1. **Fork 项目到你的 GitHub**
2. **在 Render 创建 Web Service**
3. **连接 GitHub 仓库**
4. **配置环境变量**：
   - `ARK_API_KEY`: 你的豆包 API 密钥
   - `ENVIRONMENT`: `production`
5. **自动部署完成**

### 其他平台支持

- ✅ **Railway** - 适合完整应用部署
- ✅ **Fly.io** - 全球边缘部署
- ✅ **Digital Ocean** - App Platform
- ✅ **Heroku** - 经典 PaaS 平台

## 📊 使用示例

### JavaScript/Node.js

```javascript
const API_BASE = "https://everyday-study-backend.onrender.com/api";

// 获取今日英语学习内容
async function getTodayEnglish() {
	const response = await fetch(`${API_BASE}/today-learning/english`);
	const data = await response.json();

	if (data.success) {
		console.log("今日谚语:", data.data.content);
		console.log("中文解释:", data.data.interpretation);
		console.log("关键词:", data.data.key_words);
	}
}

// 获取学习统计
async function getStats() {
	const response = await fetch(`${API_BASE}/stats`);
	const data = await response.json();
	return data.data.stats;
}
```

### Python

```python
import requests

API_BASE = "https://everyday-study-backend.onrender.com/api"

def get_today_learning(learning_type):
    """获取今日学习内容"""
    response = requests.get(f"{API_BASE}/today-learning/{learning_type}")
    return response.json()

def get_learning_history(limit=10):
    """获取学习历史"""
    response = requests.get(f"{API_BASE}/learning-history?limit={limit}")
    return response.json()

# 使用示例
english_content = get_today_learning("english")
print(english_content["data"]["content"])
```

### curl 命令

```bash
# 获取今日中医学习内容
curl -X GET "https://everyday-study-backend.onrender.com/api/today-learning/tcm" \
  -H "Accept: application/json"

# 获取英语学习历史
curl -X GET "https://everyday-study-backend.onrender.com/api/learning-history/english?limit=5"
```

## 📋 环境变量配置

```bash
# 服务器配置
PORT=91
ENVIRONMENT=production

# 数据库配置
DATABASE_PATH=learning.db

# AI API 配置（必需）
ARK_API_KEY=你的豆包API密钥
VOLCANO_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
```

## 🛡️ 安全特性

- ✅ **API 密钥保护**: 敏感信息不暴露在代码中
- ✅ **CORS 支持**: 跨域请求安全控制
- ✅ **错误处理**: 统一的错误响应格式
- ✅ **输入验证**: 参数类型和格式验证
- ✅ **Docker 安全**: 非 root 用户运行

## 📈 性能特点

- ⚡ **高并发**: Go 协程天然支持
- 🚀 **快速响应**: 平均响应时间 < 200ms
- 💾 **智能缓存**: 同一天内容缓存机制
- 📦 **轻量级**: 编译后二进制文件 < 20MB
- 🔄 **自动重启**: 服务异常自动恢复

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. **Fork 项目**
2. **创建特性分支** (`git checkout -b feature/amazing-feature`)
3. **提交更改** (`git commit -m 'Add some amazing feature'`)
4. **推送分支** (`git push origin feature/amazing-feature`)
5. **创建 Pull Request**

### 开发规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方编码规范
- 添加必要的单元测试
- 更新相关文档

## 📝 更新日志

### v1.2.0 (2024-12-24) - 当前版本

- ✨ 成功部署到 Render 云平台
- 🐳 添加完整的 Docker 支持
- 🔧 修复环境变量和端口配置问题
- 📚 完善 API 文档和使用示例
- 🌐 提供在线服务地址

### v1.1.0 (2024-12-24)

- 🚀 移除用户系统，改为全局共享
- 📦 优化数据库结构和查询性能
- 🔒 增强错误处理和安全性

### v1.0.0 (2024-12-24)

- ✨ 初始版本发布
- 🤖 集成豆包大模型 API
- 📚 支持三种学习内容类型
- 🏗️ 完整的 Go + Gin 架构

## 🆘 常见问题

### Q: 为什么有时候 API 响应很慢？

A: Render 免费版会在 15 分钟无活动后休眠，首次访问需要 30-60 秒唤醒时间。

### Q: 可以修改学习内容类型吗？

A: 目前支持 english、chinese、tcm 三种类型。如需添加新类型，请提交 Issue 或 PR。

### Q: 数据是否会丢失？

A: 云端部署的数据会持久化保存，但建议定期备份重要数据。

### Q: 如何获取豆包 API 密钥？

A: 访问 [字节跳动火山引擎](https://www.volcengine.com/product/doubao) 注册并申请 API 密钥。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源协议。

## 💡 致谢

- [Gin Web Framework](https://gin-gonic.com/) - 优秀的 Go Web 框架
- [GORM](https://gorm.io/) - 强大的 Go ORM 库
- [豆包大模型](https://www.volcengine.com/product/doubao) - 提供 AI 能力支持
- [Render](https://render.com/) - 优秀的云部署平台
- [SQLite](https://sqlite.org/) - 轻量级数据库引擎

## 📞 联系方式

- 🌐 **项目地址**: [GitHub](https://github.com/wurslu/everyday-study-backend)
- 🐛 **问题反馈**: [Issues](https://github.com/wurslu/everyday-study-backend/issues)
- 📧 **作者邮箱**: [联系方式]
- 🚀 **在线服务**: https://everyday-study-backend.onrender.com

---

⭐ **如果这个项目对你有帮助，请给个 Star 支持一下！**

🌟 **Star 数量越多，更新越频繁！**
