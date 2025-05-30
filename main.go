package main

import (
	"fmt"
	"log"
	"os"

	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/database"
	"everyday-study-backend/internal/handlers"
	"everyday-study-backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := database.Init()
	if err != nil {
		log.Fatal("数据库初始化失败:", err)
	}

	// 设置 Gin 模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.Default()

	// 配置中间件
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router.Use(middleware.ErrorHandler())

	// 创建处理器
	handler := handlers.New(db)

	// 配置路由
	api := router.Group("/api")
	{
		api.GET("/health", handler.Health)
		api.GET("/today-learning/:type", handler.GetTodayLearning)
		api.GET("/learning-history", handler.GetLearningHistory)
		api.GET("/learning-history/:type", handler.GetLearningHistoryByType)
		api.GET("/stats", handler.GetGlobalStats)
	}

	// 404 处理
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"success":    false,
			"message":    "API接口不存在",
			"error_code": "NOT_FOUND",
		})
	})

	// Render 端口配置 - 关键修复！
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}

	// 启动信息
	fmt.Println("🚀 学习助手后端服务启动成功！")
	fmt.Printf("📡 服务地址: http://0.0.0.0:%s\n", port)
	fmt.Println("📊 API文档:")
	fmt.Println("   GET  /api/health - 健康检查")
	fmt.Println("   GET  /api/today-learning/{type} - 获取今日学习内容")
	fmt.Println("   GET  /api/learning-history - 获取所有学习历史")
	fmt.Println("   GET  /api/learning-history/{type} - 获取指定类型学习历史")
	fmt.Println("   GET  /api/stats - 获取全局统计")
	fmt.Println("📚 支持的学习类型: english, chinese, tcm")
	fmt.Println("💡 注意：现在所有用户在同一天看到相同内容！")

	// 简化启动逻辑 - 移除优雅关闭的复杂性
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}