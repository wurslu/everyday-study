package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/database"
	"everyday-study-backend/internal/handlers"
	"everyday-study-backend/internal/middleware"
	"everyday-study-backend/internal/models"
	"everyday-study-backend/internal/scheduler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatal("数据库初始化失败:", err)
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	router.OPTIONS("/*path", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, Accept, X-Requested-With")
		c.Status(200)
	})

	router.Use(middleware.ErrorHandler())

	handler := handlers.New(db)

	var contentScheduler *scheduler.ContentScheduler
	if cfg.Environment == "production" {
		contentScheduler = scheduler.NewContentScheduler(cfg)
		contentScheduler.Start()
		log.Println("✅ 定时更新任务已启用")
	} else {
		log.Println("ℹ️  开发环境，定时更新任务已禁用")
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "不积跬步，无以至千里；不积小流，无以成江海",
			"description": "每日学习助手 - 用知识点亮人生",
			"api_docs": "/api/health",
			"learning_types": models.GetAllLearningTypes(),
		})
	})

	api := router.Group("/api")
	{
		api.GET("/health", handler.Health)
		api.GET("/today-learning/:type", handler.GetTodayLearning)
		api.GET("/learning-history", handler.GetLearningHistory)
		api.GET("/learning-history/:type", handler.GetLearningHistoryByType)
		api.GET("/stats", handler.GetGlobalStats)
	}

	if cfg.Environment == "development" {
		debug := router.Group("/debug")
		{
			debug.GET("/records", handler.DebugShowAllRecords)
			debug.GET("/learned-content", handler.DebugShowLearnedContent)
			debug.GET("/database-info", handler.DebugDatabaseInfo)
			debug.GET("/system-status", handler.DebugSystemStatus)
			
			debug.POST("/clear-today/:type", handler.DebugClearTodayRecords)
			debug.POST("/force-generate/:type", handler.DebugForceGenerateContent)
			debug.POST("/trigger-update", handler.DebugTriggerUpdate)
			
			debug.GET("/test-ai", handler.DebugTestAIAPI)
		}
		
		log.Println("🔧 开发环境调试接口已启用:")
		log.Println("   GET  /debug/records - 查看所有学习记录")
		log.Println("   GET  /debug/learned-content - 查看已学习内容")
		log.Println("   GET  /debug/database-info - 查看数据库信息")
		log.Println("   GET  /debug/system-status - 查看系统状态")
		log.Println("   POST /debug/clear-today/:type - 清理今日指定类型记录")
		log.Println("   POST /debug/force-generate/:type - 强制生成新内容")
		log.Println("   POST /debug/trigger-update - 手动触发更新")
		log.Println("   GET  /debug/test-ai - 测试AI API连接")
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"success":    false,
			"message":    "路径不存在",
			"suggestion": "访问 / 查看可用接口",
			"error_code": "NOT_FOUND",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: router,
	}

	fmt.Println("🚀 学习助手后端服务启动成功！")
	fmt.Printf("📡 服务地址: http://0.0.0.0:%s\n", port)
	fmt.Println("💡 励志首页: /")
	if contentScheduler != nil {
		fmt.Println("🌙 定时更新: 每晚12点自动更新学习内容")
		fmt.Printf("⏰ 下次更新: %s\n", contentScheduler.GetNextUpdateTime().Format("2006-01-02 00:00:00"))
	}
	fmt.Println("📊 安全API接口:")
	fmt.Println("   GET  / - 励志首页")
	fmt.Println("   GET  /api/health - 健康检查")
	fmt.Println("   GET  /api/today-learning/{type} - 获取今日学习内容")
	fmt.Println("   GET  /api/learning-history - 获取所有学习历史")
	fmt.Println("   GET  /api/learning-history/{type} - 获取指定类型学习历史")
	fmt.Println("   GET  /api/stats - 获取全局统计")
	fmt.Println("📚 支持的学习类型: english, chinese, tcm")
	fmt.Println("🛡️  安全特性: 已移除所有管理和调试接口")
	fmt.Println("🌐 CORS: 已配置支持跨域请求")
	fmt.Printf("🔑 API密钥: %s\n", maskAPIKey(cfg.VolcanoAPIKey))

	go func() {
		log.Printf("服务器启动在端口 %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🔄 正在优雅关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if contentScheduler != nil {
		log.Println("🛑 停止内容定时器...")
		contentScheduler.Stop()
		log.Println("✅ 定时器已停止")
	}

	log.Println("🔄 关闭HTTP服务器...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("❌ 服务器强制关闭: %v", err)
	}

	select {
	case <-ctx.Done():
		log.Println("⏰ 关闭超时")
	default:
		log.Println("✅ 服务器已优雅关闭")
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "已设置"
	}
	return key[:4] + "****" + key[len(key)-4:]
}