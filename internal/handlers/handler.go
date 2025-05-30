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

	// 检查今日是否已有记录
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

	// 如果今日已有记录，直接返回
	if todayRecord != nil {
		log.Printf("返回今日已缓存的%s内容", models.GetLearningTypeName(learningType))
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

	// 生成新内容
	log.Printf("获取新的 %s 学习内容...", models.GetLearningTypeName(learningType))

	// 获取已学习内容列表
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

	// 调用AI API
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
	log.Printf("AI原始响应: %s", content)

	// 使用改进的解析方法
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

	// 创建学习内容
	learningContent := models.LearningContent{
		Type:           models.LearningType(learningType),
		Content:        parsedContent.Content,
		Interpretation: parsedContent.Interpretation,
		KeyWords:       parsedContent.KeyWords,
		Date:           time.Now(),
	}

	log.Printf("创建的学习内容: %+v", learningContent)

	// 保存到数据库
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

	log.Printf("成功保存%s学习记录, ID: %d", models.GetLearningTypeName(learningType), savedRecord.ID)

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

// 定义统一的解析结果结构
type ParsedContent struct {
	Content        string
	Interpretation string
	KeyWords       []string
}

// 改进的AI内容解析方法
func (h *Handler) parseAIContent(contentStr string, learningType string) (*ParsedContent, error) {
	// 清理可能的markdown代码块标记
	contentStr = strings.TrimPrefix(contentStr, "```json")
	contentStr = strings.TrimSuffix(contentStr, "```")
	contentStr = strings.TrimSpace(contentStr)

	// 先尝试原有的解析方式
	var aiData models.AIContent
	err := json.Unmarshal([]byte(contentStr), &aiData)
	if err == nil {
		// 验证并提取内容
		result := h.extractContentFromAIData(&aiData, learningType)
		if result != nil {
			return result, nil
		}
	}

	log.Printf("直接解析失败，尝试灵活解析: %v", err)

	// 尝试灵活解析
	return h.flexibleParseContent(contentStr, learningType)
}

// 从AIData中提取内容
func (h *Handler) extractContentFromAIData(aiData *models.AIContent, learningType string) *ParsedContent {
	result := &ParsedContent{
		Interpretation: aiData.Interpretation,
		KeyWords:       []string{},
	}

	// 根据类型提取主内容
	switch strings.ToLower(learningType) {
	case "english":
		if aiData.Proverb != "" {
			result.Content = aiData.Proverb
		}
		// 转换KeyWords
		if len(aiData.KeyWords) > 0 {
			for _, kw := range aiData.KeyWords {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kw.Word, kw.Meaning))
			}
		}
	case "chinese":
		if aiData.Poem != "" {
			result.Content = aiData.Poem
		}
		// 转换KeyWords
		if len(aiData.KeyWords) > 0 {
			for _, kw := range aiData.KeyWords {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kw.Word, kw.Meaning))
			}
		}
	case "tcm":
		if aiData.TCMText != "" {
			result.Content = aiData.TCMText
		}
		// 转换KeyConcepts
		if len(aiData.KeyConcepts) > 0 {
			for _, kc := range aiData.KeyConcepts {
				result.KeyWords = append(result.KeyWords, fmt.Sprintf("%s: %s", kc.Concept, kc.Meaning))
			}
		}
	}

	// 验证必要字段
	if result.Content == "" || result.Interpretation == "" {
		return nil
	}

	return result
}

// 灵活解析内容
func (h *Handler) flexibleParseContent(contentStr string, learningType string) (*ParsedContent, error) {
	// 使用map来灵活解析
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

// 安全获取字符串值
func (h *Handler) getStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// 解析关键项目
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
					// 如果直接是字符串
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
