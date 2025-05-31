package handlers

import (
	"encoding/json"
	"everyday-study-backend/internal/api"
	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/database"
	"everyday-study-backend/internal/models"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db            *gorm.DB
	volcanoClient *api.VolcanoClient
}

func New(db *gorm.DB) *Handler {
	cfg := config.Load()
	return &Handler{
		db:            db,
		volcanoClient: api.NewVolcanoClient(cfg),
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "服务运行正常",
		Data: models.HealthData{
			Status:         "ok",
			Database:       "connected",
			SupportedTypes: models.GetAllLearningTypes(),
		},
	})
}

func (h *Handler) GetTodayLearning(c *gin.Context) {
	learningType := c.Param("type")

	if !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
			Errors:    []string{fmt.Sprintf("支持的类型: %s", strings.Join(models.GetAllLearningTypes(), ", "))},
		})
		return
	}

	fmt.Printf("📥 收到请求 - 类型: %s, 时间: %s\n", 
		models.GetLearningTypeName(learningType), 
		time.Now().Format("2006-01-02 15:04:05"))

	todayRecord, err := database.GetTodayLearningRecord(learningType)
	if err != nil {
		log.Printf("获取今日学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取今日学习记录失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	if todayRecord != nil {
		fmt.Printf("🎯 返回今日已缓存的%s内容，记录ID: %d\n", 
			models.GetLearningTypeName(learningType), todayRecord.ID)
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Message: "获取今日学习内容成功",
			Data: models.TodayLearningData{
				Type:           todayRecord.Type,
				TypeName:       models.GetLearningTypeName(todayRecord.Type),
				Content:        todayRecord.Content,
				Interpretation: todayRecord.Interpretation,
				KeyWords:       todayRecord.FormatKeyWords(),
				Date:           todayRecord.Date.Format("2006-01-02"),
				FromCache:      true,
			},
		})
		return
	}

	fmt.Printf("🆕 今日尚无%s内容，开始生成新内容...\n", models.GetLearningTypeName(learningType))

	learnedContent, err := database.GetLearnedContent(learningType)
	if err != nil {
		log.Printf("获取已学习内容失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取已学习内容失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	fmt.Printf("📚 已学习内容数量: %d\n", len(learnedContent))

	aiResponse, err := h.volcanoClient.CallVolcanoAPI(learningType, learnedContent)
	if err != nil {
		log.Printf("调用AI API失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   fmt.Sprintf("调用AI API失败: %s", err.Error()),
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	if len(aiResponse.Choices) == 0 {
		log.Printf("AI API返回空响应")
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "AI API返回空响应",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	content := aiResponse.Choices[0].Message.Content
	log.Printf("🤖 AI原始响应: %s", content[:min(100, len(content))]+"...")

	parsedContent, err := h.parseAIContent(content, learningType)
	if err != nil {
		log.Printf("解析AI内容失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   fmt.Sprintf("解析AI内容失败: %s", err.Error()),
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	learningContent := models.LearningContent{
		Type:           models.LearningType(learningType),
		Content:        parsedContent.Content,
		Interpretation: parsedContent.Interpretation,
		KeyWords:       parsedContent.KeyWords,
		Date:           time.Now(),
	}

	log.Printf("📝 创建的学习内容: %+v", learningContent)

	savedRecord, err := database.SaveLearningRecord(learningType, learningContent)
	if err != nil {
		log.Printf("保存学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   fmt.Sprintf("保存学习记录失败: %s", err.Error()),
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	fmt.Printf("✅ 成功保存%s学习记录, ID: %d\n", models.GetLearningTypeName(learningType), savedRecord.ID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取今日学习内容成功",
		Data: models.TodayLearningData{
			Type:           savedRecord.Type,
			TypeName:       models.GetLearningTypeName(savedRecord.Type),
			Content:        savedRecord.Content,
			Interpretation: savedRecord.Interpretation,
			KeyWords:       savedRecord.FormatKeyWords(),
			Date:           savedRecord.Date.Format("2006-01-02"),
			FromCache:      false,
		},
	})
}

type ParsedContent struct {
	Content        string
	Interpretation string
	KeyWords       []string
}

func (h *Handler) parseAIContent(contentStr string, learningType string) (*ParsedContent, error) {
	contentStr = strings.TrimPrefix(contentStr, "```json")
	contentStr = strings.TrimSuffix(contentStr, "```")
	contentStr = strings.TrimSpace(contentStr)

	var aiData models.AIContent
	err := json.Unmarshal([]byte(contentStr), &aiData)
	if err == nil {
		result := h.extractContentFromAIData(&aiData, learningType)
		if result != nil {
			return result, nil
		}
	}

	log.Printf("直接解析失败，尝试灵活解析: %v", err)

	return h.flexibleParseContent(contentStr, learningType)
}

func (h *Handler) extractContentFromAIData(aiData *models.AIContent, learningType string) *ParsedContent {
	result := &ParsedContent{
		Interpretation: aiData.Interpretation,
		KeyWords:       []string{},
	}

	switch strings.ToLower(learningType) {
	case "english":
		if aiData.Proverb != "" {
			result.Content = aiData.Proverb
		}
		if len(aiData.KeyWords) > 0 {
			for _, kw := range aiData.KeyWords {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kw.Word, kw.Meaning))
			}
		}
	case "chinese":
		if aiData.Poem != "" {
			result.Content = aiData.Poem
		}
		if len(aiData.KeyWords) > 0 {
			for _, kw := range aiData.KeyWords {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kw.Word, kw.Meaning))
			}
		}
	case "tcm":
		if aiData.TCMText != "" {
			result.Content = aiData.TCMText
		}
		if len(aiData.KeyConcepts) > 0 {
			for _, kc := range aiData.KeyConcepts {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kc.Concept, kc.Meaning))
			}
		}
	}

	if result.Content == "" || result.Interpretation == "" {
		return nil
	}

	return result
}

func (h *Handler) flexibleParseContent(contentStr string, learningType string) (*ParsedContent, error) {
	var rawContent map[string]interface{}
	err := json.Unmarshal([]byte(contentStr), &rawContent)
	if err != nil {
		return nil, fmt.Errorf("无法解析JSON内容: %v", err)
	}

	result := &ParsedContent{
		KeyWords: []string{},
	}

	switch strings.ToLower(learningType) {
	case "english":
		result.Content = h.getStringValue(rawContent, "proverb")
		result.Interpretation = h.getStringValue(rawContent, "interpretation")
		result.KeyWords = h.parseKeyItems(rawContent, "key_words", "word", "meaning")

	case "chinese":
		result.Content = h.getStringValue(rawContent, "poem")
		result.Interpretation = h.getStringValue(rawContent, "interpretation")
		result.KeyWords = h.parseKeyItems(rawContent, "key_words", "word", "meaning")

	case "tcm":
		result.Content = h.getStringValue(rawContent, "tcm_text")
		result.Interpretation = h.getStringValue(rawContent, "interpretation")
		result.KeyWords = h.parseKeyItems(rawContent, "key_concepts", "concept", "meaning")
	}

	if result.Content == "" {
		return nil, fmt.Errorf("解析后的主要内容为空")
	}

	if result.Interpretation == "" {
		return nil, fmt.Errorf("解析后的释义为空")
	}

	return result, nil
}

func (h *Handler) getStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (h *Handler) parseKeyItems(data map[string]interface{}, arrayKey, itemKey, meaningKey string) []string {
	var result []string

	if value, exists := data[arrayKey]; exists {
		if array, ok := value.([]interface{}); ok {
			for _, item := range array {
				if itemMap, ok := item.(map[string]interface{}); ok {
					itemValue := h.getStringValue(itemMap, itemKey)
					meaningValue := h.getStringValue(itemMap, meaningKey)
					if itemValue != "" && meaningValue != "" {
						result = append(result, fmt.Sprintf("%s: %s", itemValue, meaningValue))
					}
				} else if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
		}
	}

	return result
}

func (h *Handler) GetLearningHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	records, err := database.GetLearningHistory("", limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取学习历史失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	historyItems := make([]models.LearningHistoryItem, len(records))
	for i, record := range records {
		historyItems[i] = models.LearningHistoryItem{
			Type:           record.Type,
			TypeName:       models.GetLearningTypeName(record.Type),
			Content:        record.Content,
			Interpretation: record.Interpretation,
			KeyWords:       record.FormatKeyWords(),
			Date:           record.Date.Format("2006-01-02"),
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取学习历史成功",
		Data: models.LearningHistoryData{
			Total:   len(historyItems),
			Records: historyItems,
		},
	})
}

func (h *Handler) GetLearningHistoryByType(c *gin.Context) {
	learningType := c.Param("type")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	if !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
			Errors:    []string{fmt.Sprintf("支持的类型: %s", strings.Join(models.GetAllLearningTypes(), ", "))},
		})
		return
	}

	records, err := database.GetLearningHistory(learningType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取学习历史失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	historyItems := make([]models.LearningHistoryItem, len(records))
	for i, record := range records {
		historyItems[i] = models.LearningHistoryItem{
			Type:           record.Type,
			TypeName:       models.GetLearningTypeName(record.Type),
			Content:        record.Content,
			Interpretation: record.Interpretation,
			KeyWords:       record.FormatKeyWords(),
			Date:           record.Date.Format("2006-01-02"),
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取学习历史成功",
		Data: models.LearningHistoryData{
			Total:   len(historyItems),
			Records: historyItems,
		},
	})
}

func (h *Handler) GetGlobalStats(c *gin.Context) {
	stats, err := database.GetGlobalStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取统计信息失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取统计信息成功",
		Data: models.UserStatsData{
			Stats: stats,
		},
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *Handler) DebugShowAllRecords(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var records []models.LearningRecord
	err := h.db.Order("date DESC").Limit(limit).Find(&records).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取记录失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	debugRecords := make([]gin.H, len(records))
	for i, record := range records {
		debugRecords[i] = gin.H{
			"id":             record.ID,
			"type":           record.Type,
			"type_name":      models.GetLearningTypeName(record.Type),
			"content":        record.Content,
			"interpretation": record.Interpretation,
			"key_words":      record.FormatKeyWords(),
			"date":           record.Date.Format("2006-01-02 15:04:05"),
			"created_at":     record.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取所有记录成功",
		Data: gin.H{
			"total":   len(debugRecords),
			"records": debugRecords,
		},
	})
}

func (h *Handler) DebugClearTodayRecords(c *gin.Context) {
	learningType := c.Param("type")

	if !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
		})
		return
	}

	database.DebugClearTodayRecords(learningType)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "清理今日记录成功",
		Data: gin.H{
			"type":      learningType,
			"type_name": models.GetLearningTypeName(learningType),
			"action":    "清理今日记录",
		},
	})
}

func (h *Handler) DebugForceGenerateContent(c *gin.Context) {
	learningType := c.Param("type")

	if !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
		})
		return
	}

	database.DebugClearTodayRecords(learningType)

	h.GetTodayLearning(c)
}

func (h *Handler) DebugShowLearnedContent(c *gin.Context) {
	learningType := c.Query("type")

	if learningType != "" && !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
		})
		return
	}

	result := make(map[string]interface{})

	if learningType != "" {
		contents, err := database.GetLearnedContent(learningType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success:   false,
				Message:   "获取已学习内容失败",
				ErrorCode: "SERVER_ERROR",
			})
			return
		}

		result[learningType] = gin.H{
			"type_name": models.GetLearningTypeName(learningType),
			"count":     len(contents),
			"contents":  contents,
		}
	} else {
		allTypes := models.GetAllLearningTypes()
		for _, t := range allTypes {
			contents, err := database.GetLearnedContent(t)
			if err != nil {
				result[t] = gin.H{
					"type_name": models.GetLearningTypeName(t),
					"count":     0,
					"error":     err.Error(),
				}
			} else {
				result[t] = gin.H{
					"type_name": models.GetLearningTypeName(t),
					"count":     len(contents),
					"contents":  contents,
				}
			}
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取已学习内容成功",
		Data:    result,
	})
}

func (h *Handler) DebugDatabaseInfo(c *gin.Context) {
	var learningRecordCount int64
	var learnedContentCount int64

	h.db.Model(&models.LearningRecord{}).Count(&learningRecordCount)
	h.db.Model(&models.LearnedContent{}).Count(&learnedContentCount)

	typeStats := make(map[string]gin.H)
	for _, t := range models.GetAllLearningTypes() {
		var recordCount int64
		var contentCount int64

		h.db.Model(&models.LearningRecord{}).Where("type = ?", t).Count(&recordCount)
		h.db.Model(&models.LearnedContent{}).Where("type = ?", t).Count(&contentCount)

		typeStats[t] = gin.H{
			"type_name":      models.GetLearningTypeName(t),
			"record_count":   recordCount,
			"content_count":  contentCount,
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "获取数据库信息成功",
		Data: gin.H{
			"total_learning_records": learningRecordCount,
			"total_learned_contents": learnedContentCount,
			"type_statistics":        typeStats,
		},
	})
}

func (h *Handler) DebugTestAIAPI(c *gin.Context) {
	learningType := c.DefaultQuery("type", "english")

	if !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
		})
		return
	}

	learnedContent, err := database.GetLearnedContent(learningType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取已学习内容失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	testLearned := learnedContent
	if len(testLearned) > 5 {
		testLearned = testLearned[:5]
	}

	aiResponse, err := h.volcanoClient.CallVolcanoAPI(learningType, testLearned)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "AI API调用失败",
			ErrorCode: "AI_API_ERROR",
			Errors:    []string{err.Error()},
		})
		return
	}

	if len(aiResponse.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "AI API返回空响应",
			ErrorCode: "AI_API_ERROR",
		})
		return
	}

	content := aiResponse.Choices[0].Message.Content

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "AI API测试成功",
		Data: gin.H{
			"type":              learningType,
			"type_name":         models.GetLearningTypeName(learningType),
			"learned_count":     len(testLearned),
			"test_learned":      testLearned,
			"ai_raw_response":   content,
			"response_length":   len(content),
		},
	})
}

func (h *Handler) DebugTriggerUpdate(c *gin.Context) {
	learningType := c.Query("type")

	if learningType != "" && !models.IsValidLearningType(learningType) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "无效的学习类型",
			ErrorCode: "VALIDATION_ERROR",
		})
		return
	}

	if learningType != "" {
		database.DebugClearTodayRecords(learningType)
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Message: "已清理指定类型的今日记录，请重新访问对应的学习接口",
			Data: gin.H{
				"type":      learningType,
				"type_name": models.GetLearningTypeName(learningType),
				"action":    "清理今日记录",
				"next_step": "访问 /api/today-learning/" + learningType,
			},
		})
	} else {
		for _, t := range models.GetAllLearningTypes() {
			database.DebugClearTodayRecords(t)
		}
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Message: "已清理所有类型的今日记录",
			Data: gin.H{
				"action":    "清理所有今日记录",
				"types":     models.GetAllLearningTypes(),
				"next_step": "重新访问各类型的学习接口将生成新内容",
			},
		})
	}
}

func (h *Handler) DebugSystemStatus(c *gin.Context) {
	sqlDB, err := h.db.DB()
	var dbStatus string
	if err != nil {
		dbStatus = "获取连接失败: " + err.Error()
	} else {
		err = sqlDB.Ping()
		if err != nil {
			dbStatus = "连接测试失败: " + err.Error()
		} else {
			dbStatus = "连接正常"
		}
	}

	todayStatus := make(map[string]gin.H)
	for _, t := range models.GetAllLearningTypes() {
		record, err := database.GetTodayLearningRecord(t)
		if err != nil {
			todayStatus[t] = gin.H{
				"type_name": models.GetLearningTypeName(t),
				"status":    "检查失败",
				"error":     err.Error(),
			}
		} else if record == nil {
			todayStatus[t] = gin.H{
				"type_name": models.GetLearningTypeName(t),
				"status":    "今日无记录",
				"record":    nil,
			}
		} else {
			todayStatus[t] = gin.H{
				"type_name": models.GetLearningTypeName(t),
				"status":    "今日已有记录",
				"record_id": record.ID,
				"date":      record.Date.Format("2006-01-02 15:04:05"),
				"content":   record.Content[:min(50, len(record.Content))] + "...",
			}
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "系统状态检查完成",
		Data: gin.H{
			"database_status":    dbStatus,
			"today_content_status": todayStatus,
			"supported_types":    models.GetAllLearningTypes(),
		},
	})
}