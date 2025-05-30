package database

import (
	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/models"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) (*gorm.DB, error) {
	var err error
	
	// 使用配置文件中的数据库路径
	DB, err = gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	err = DB.AutoMigrate(
		&models.LearningRecord{},
		&models.LearnedContent{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	fmt.Println("✅ 数据库初始化完成")
	return DB, nil
}

func GetLearnedContent(learningType string) ([]string, error) {
	var contents []models.LearnedContent
	
	err := DB.Where("type = ?", learningType).
		Order("created_at").
		Find(&contents).Error
	
	if err != nil {
		return nil, fmt.Errorf("获取已学习内容失败: %v", err)
	}

	var result []string
	for _, content := range contents {
		result = append(result, content.Content)
	}

	return result, nil
}

// 修复：获取今日学习记录的函数
func GetTodayLearningRecord(learningType string) (*models.LearningRecord, error) {
	var record models.LearningRecord
	
	// 获取今天的开始和结束时间（本地时间）
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	
	fmt.Printf("🔍 查询今日记录 - 类型: %s, 时间范围: %s 到 %s\n", 
		learningType, 
		todayStart.Format("2006-01-02 15:04:05"), 
		todayEnd.Format("2006-01-02 15:04:05"))
	
	err := DB.Where("type = ? AND date >= ? AND date < ?", learningType, todayStart, todayEnd).
		Order("date DESC").
		First(&record).Error
		
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("📝 今日暂无 %s 学习记录\n", models.GetLearningTypeName(learningType))
			return nil, nil
		}
		return nil, fmt.Errorf("获取今日学习记录失败: %v", err)
	}

	fmt.Printf("✅ 找到今日 %s 学习记录，ID: %d, 创建时间: %s\n", 
		models.GetLearningTypeName(learningType), 
		record.ID, 
		record.Date.Format("2006-01-02 15:04:05"))

	return &record, nil
}

// 修复：保存学习记录的函数
func SaveLearningRecord(learningType string, content models.LearningContent) (*models.LearningRecord, error) {
	if errors := content.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("数据验证失败: %v", errors)
	}

	now := time.Now()
	// 获取今天的开始和结束时间
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	fmt.Printf("💾 保存学习记录 - 类型: %s, 时间: %s\n", 
		learningType, now.Format("2006-01-02 15:04:05"))

	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除今天同类型的旧记录（如果有的话）
	var deleteCount int64
	result := tx.Where("type = ? AND date >= ? AND date < ?", learningType, todayStart, todayEnd).
		Delete(&models.LearningRecord{})
	if result.Error != nil {
		tx.Rollback()
		return nil, fmt.Errorf("删除旧记录失败: %v", result.Error)
	}
	deleteCount = result.RowsAffected
	
	if deleteCount > 0 {
		fmt.Printf("🗑️  删除了 %d 条今日旧记录\n", deleteCount)
	}

	// 创建新记录
	record := models.LearningRecord{
		Type:           learningType,
		Content:        content.Content,
		Interpretation: content.Interpretation,
		KeyWords:       content.FormatKeyWords(),
		Date:           now, // 使用当前完整时间
	}

	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("保存学习记录失败: %v", err)
	}

	fmt.Printf("✅ 学习记录已保存，ID: %d\n", record.ID)

	// 保存到已学习内容表（防重复）
	learnedContent := models.LearnedContent{
		Type:    learningType,
		Content: content.Content,
	}
	
	var existing models.LearnedContent
	if err := tx.Where("type = ? AND content = ?", learningType, content.Content).
		FirstOrCreate(&existing, learnedContent).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("保存已学习内容失败: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("事务提交失败: %v", err)
	}

	return &record, nil
}

func GetLearningHistory(learningType string, limit int) ([]models.LearningRecord, error) {
	var records []models.LearningRecord
	
	query := DB.Model(&models.LearningRecord{})
	if learningType != "" {
		query = query.Where("type = ?", learningType)
	}
	
	err := query.Order("date DESC").
		Limit(limit).
		Find(&records).Error
		
	if err != nil {
		return nil, fmt.Errorf("获取学习历史失败: %v", err)
	}

	return records, nil
}

func GetGlobalStats() (map[string]models.TypeStats, error) {
	type StatResult struct {
		Type       string `json:"type"`
		TotalDays  int64  `json:"total_days"`
		UniqueDays int64  `json:"unique_days"`
	}

	var results []StatResult
	
	err := DB.Model(&models.LearningRecord{}).
		Select("type, COUNT(*) as total_days, COUNT(DISTINCT DATE(date)) as unique_days").
		Group("type").
		Find(&results).Error
		
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %v", err)
	}

	stats := make(map[string]models.TypeStats)
	for _, result := range results {
		stats[result.Type] = models.TypeStats{
			TypeName:   models.GetLearningTypeName(result.Type),
			TotalDays:  int(result.TotalDays),
			UniqueDays: int(result.UniqueDays),
		}
	}

	return stats, nil
}

// 添加到 internal/database/database.go 文件末尾
func DebugShowAllRecords() {
    var records []models.LearningRecord
    DB.Order("date DESC").Limit(20).Find(&records)
    
    fmt.Println("\n📊 最近20条学习记录:")
    fmt.Println("ID | 类型 | 日期 | 内容预览")
    fmt.Println("---|------|------|--------")
    for _, record := range records {
        content := record.Content
        if len(content) > 30 {
            content = content[:30] + "..."
        }
        fmt.Printf("%d | %s | %s | %s\n", 
            record.ID,
            models.GetLearningTypeName(record.Type),
            record.Date.Format("2006-01-02 15:04"),
            content)
    }
    fmt.Println()
}

func DebugClearTodayRecords(learningType string) {
    now := time.Now()
    todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    todayEnd := todayStart.Add(24 * time.Hour)
    
    result := DB.Where("type = ? AND date >= ? AND date < ?", learningType, todayStart, todayEnd).
        Delete(&models.LearningRecord{})
    
    fmt.Printf("🗑️  已清理今日 %s 记录，删除了 %d 条\n", 
        models.GetLearningTypeName(learningType), result.RowsAffected)
}