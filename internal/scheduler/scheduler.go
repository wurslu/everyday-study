package scheduler

import (
	"encoding/json"
	"everyday-study-backend/internal/api"
	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/database"
	"everyday-study-backend/internal/models"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type ContentScheduler struct {
	volcanoClient *api.VolcanoClient
	ticker        *time.Ticker
	quit          chan bool
	wg            sync.WaitGroup
	running       bool
	mu            sync.Mutex
}

type ParsedContent struct {
	Content        string
	Interpretation string
	KeyWords       []string
}

func NewContentScheduler(cfg *config.Config) *ContentScheduler {
	return &ContentScheduler{
		volcanoClient: api.NewVolcanoClient(cfg),
		quit:         make(chan bool, 1),
		running:      false,
	}
}

func (cs *ContentScheduler) Start() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.running {
		log.Println("⚠️  定时器已经在运行中")
		return
	}
	
	cs.running = true
	log.Println("🌙 内容定时器启动 - 每晚12点更新...")
	
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	timeUntilMidnight := time.Until(nextMidnight)
	
	log.Printf("⏰ 下次内容更新时间: %s (还有 %v)", 
		nextMidnight.Format("2006-01-02 00:00:00"), timeUntilMidnight)
	
	cs.wg.Add(1)
	go func() {
		defer cs.wg.Done()
		
		timer := time.NewTimer(timeUntilMidnight)
		
		select {
		case <-timer.C:
			log.Println("🕛 午夜12点到了，开始更新内容...")
			cs.updateAllContent()
			
			cs.ticker = time.NewTicker(24 * time.Hour)
			
			for {
				select {
				case <-cs.ticker.C:
					log.Println("🕛 每日定时更新开始...")
					cs.updateAllContent()
				case <-cs.quit:
					log.Println("📨 收到退出信号，停止定时任务")
					return
				}
			}
		case <-cs.quit:
			timer.Stop()
			log.Println("📨 收到退出信号，取消首次等待")
			return
		}
	}()
}

func (cs *ContentScheduler) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if !cs.running {
		log.Println("ℹ️  定时器未运行")
		return
	}
	
	cs.running = false
	
	select {
	case cs.quit <- true:
		log.Println("📤 已发送退出信号")
	default:
		log.Println("⚠️  退出信号通道已满或已关闭")
	}
	
	if cs.ticker != nil {
		cs.ticker.Stop()
		log.Println("⏹️  Ticker已停止")
	}
	
	done := make(chan bool, 1)
	go func() {
		cs.wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		log.Println("✅ 所有定时任务已停止")
	case <-time.After(3 * time.Second):
		log.Println("⏰ 等待定时任务停止超时，强制退出")
	}
}

func (cs *ContentScheduler) updateAllContent() {
	cs.wg.Add(1)
	defer cs.wg.Done()
	
	log.Println("🔄 开始定时更新学习内容...")
	startTime := time.Now()
	
	learningTypes := models.GetAllLearningTypes()
	successCount := 0
	
	for _, learningType := range learningTypes {
		select {
		case <-cs.quit:
			log.Println("📨 更新过程中收到退出信号，停止更新")
			return
		default:
		}
		
		err := cs.updateContentForType(learningType)
		if err != nil {
			log.Printf("❌ 更新 %s 内容失败: %v", models.GetLearningTypeName(learningType), err)
		} else {
			log.Printf("✅ 成功更新 %s 内容", models.GetLearningTypeName(learningType))
			successCount++
		}
		
		time.Sleep(3 * time.Second)
	}
	
	duration := time.Since(startTime)
	log.Printf("🎉 内容更新完成！成功 %d/%d，耗时: %v", 
		successCount, len(learningTypes), duration)
}

func (cs *ContentScheduler) updateContentForType(learningType string) error {
	log.Printf("📚 正在更新 %s...", models.GetLearningTypeName(learningType))
	
	learnedContent, err := database.GetLearnedContent(learningType)
	if err != nil {
		return fmt.Errorf("获取已学习内容失败: %v", err)
	}
	
	log.Printf("📝 已学习内容数量: %d", len(learnedContent))
	
	aiResponse, err := cs.volcanoClient.CallVolcanoAPI(learningType, learnedContent)
	if err != nil {
		return fmt.Errorf("调用AI API失败: %v", err)
	}
	
	if len(aiResponse.Choices) == 0 {
		return fmt.Errorf("AI API返回空响应")
	}
	
	content := aiResponse.Choices[0].Message.Content
	log.Printf("🤖 AI原始响应: %s", content[:min(100, len(content))]+"...")
	
	parsedContent, err := cs.parseAIContent(content, learningType)
	if err != nil {
		return fmt.Errorf("解析AI内容失败: %v", err)
	}
	
	learningContent := models.LearningContent{
		Type:           models.LearningType(learningType),
		Content:        parsedContent.Content,
		Interpretation: parsedContent.Interpretation,
		KeyWords:       parsedContent.KeyWords,
		Date:           time.Now(),
	}
	
	_, err = database.SaveLearningRecord(learningType, learningContent)
	if err != nil {
		return fmt.Errorf("保存学习记录失败: %v", err)
	}
	
	return nil
}

func (cs *ContentScheduler) parseAIContent(contentStr string, learningType string) (*ParsedContent, error) {
	contentStr = strings.TrimPrefix(contentStr, "```json")
	contentStr = strings.TrimSuffix(contentStr, "```")
	contentStr = strings.TrimSpace(contentStr)

	var aiData models.AIContent
	err := json.Unmarshal([]byte(contentStr), &aiData)
	if err == nil {
		result := cs.extractContentFromAIData(&aiData, learningType)
		if result != nil {
			return result, nil
		}
	}

	log.Printf("直接解析失败，尝试灵活解析: %v", err)

	return cs.flexibleParseContent(contentStr, learningType)
}

func (cs *ContentScheduler) extractContentFromAIData(aiData *models.AIContent, learningType string) *ParsedContent {
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

func (cs *ContentScheduler) flexibleParseContent(contentStr string, learningType string) (*ParsedContent, error) {
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
		result.Content = cs.getStringValue(rawContent, "proverb")
		result.Interpretation = cs.getStringValue(rawContent, "interpretation")
		result.KeyWords = cs.parseKeyItems(rawContent, "key_words", "word", "meaning")

	case "chinese":
		result.Content = cs.getStringValue(rawContent, "poem")
		result.Interpretation = cs.getStringValue(rawContent, "interpretation")
		result.KeyWords = cs.parseKeyItems(rawContent, "key_words", "word", "meaning")

	case "tcm":
		result.Content = cs.getStringValue(rawContent, "tcm_text")
		result.Interpretation = cs.getStringValue(rawContent, "interpretation")
		result.KeyWords = cs.parseKeyItems(rawContent, "key_concepts", "concept", "meaning")
	}

	if result.Content == "" {
		return nil, fmt.Errorf("解析后的主要内容为空")
	}

	if result.Interpretation == "" {
		return nil, fmt.Errorf("解析后的释义为空")
	}

	return result, nil
}

func (cs *ContentScheduler) getStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (cs *ContentScheduler) parseKeyItems(data map[string]interface{}, arrayKey, itemKey, meaningKey string) []string {
	var result []string

	if value, exists := data[arrayKey]; exists {
		if array, ok := value.([]interface{}); ok {
			for _, item := range array {
				if itemMap, ok := item.(map[string]interface{}); ok {
					itemValue := cs.getStringValue(itemMap, itemKey)
					meaningValue := cs.getStringValue(itemMap, meaningKey)
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

func (cs *ContentScheduler) TriggerUpdate() {
	log.Println("🔧 手动触发内容更新...")
	go cs.updateAllContent()
}

func (cs *ContentScheduler) GetNextUpdateTime() time.Time {
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return nextMidnight
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}