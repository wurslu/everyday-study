package handlers

import (
	"encoding/json"
	"everyday-study-backend/internal/api"
	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/database"
	"everyday-study-backend/internal/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db           *gorm.DB
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

	todayRecord, err := database.GetTodayLearningRecord(learningType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取今日学习记录失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	if todayRecord != nil {
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

	fmt.Printf("获取新的 %s 学习内容...\n", models.GetLearningTypeName(learningType))

	learnedContent, err := database.GetLearnedContent(learningType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "获取已学习内容失败",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	aiResponse, err := h.volcanoClient.CallVolcanoAPI(learningType, learnedContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   fmt.Sprintf("调用AI API失败: %s", err.Error()),
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	content := aiResponse.Choices[0].Message.Content
	fmt.Println("AI原始响应:", content)

	var aiData models.AIContent
	if err := json.Unmarshal([]byte(content), &aiData); err != nil {
		fmt.Printf("JSON解析失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "AI返回的内容格式不正确",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	contentText := ""
	if aiData.Proverb != "" {
		contentText = aiData.Proverb
	} else if aiData.Poem != "" {
		contentText = aiData.Poem
	} else if aiData.TCMText != "" {
		contentText = aiData.TCMText
	}

	if contentText == "" {
		fmt.Printf("AI数据结构: %+v\n", aiData)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "AI返回的数据中缺少主要内容字段",
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

	fmt.Println("提取的内容文本:", contentText)

	var keyWords []string
	if len(aiData.KeyWords) > 0 {
		for _, kw := range aiData.KeyWords {
			keyWords = append(keyWords, fmt.Sprintf("%s: %s", kw.Word, kw.Meaning))
		}
	} else if len(aiData.KeyConcepts) > 0 {
		for _, kc := range aiData.KeyConcepts {
			keyWords = append(keyWords, fmt.Sprintf("%s: %s", kc.Concept, kc.Meaning))
		}
	}

	learningContent := models.LearningContent{
		Type:           models.LearningType(learningType),
		Content:        contentText,
		Interpretation: aiData.Interpretation,
		KeyWords:       keyWords,
		Date:           time.Now(),
	}

	fmt.Printf("创建的学习内容: %+v\n", learningContent)

	savedRecord, err := database.SaveLearningRecord(learningType, learningContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   fmt.Sprintf("保存学习记录失败: %s", err.Error()),
			ErrorCode: "SERVER_ERROR",
		})
		return
	}

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