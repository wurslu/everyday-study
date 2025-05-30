package database

import (
	"everyday-study-backend/internal/models"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() (*gorm.DB, error) {
	var err error
	
	DB, err = gorm.Open(sqlite.Open("learning.db"), &gorm.Config{
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

func GetTodayLearningRecord(learningType string) (*models.LearningRecord, error) {
	var record models.LearningRecord
	today := time.Now().Format("2006-01-02")
	
	err := DB.Where("type = ? AND DATE(date) = ?", learningType, today).
		First(&record).Error
		
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取今日学习记录失败: %v", err)
	}

	return &record, nil
}

func SaveLearningRecord(learningType string, content models.LearningContent) (*models.LearningRecord, error) {
	if errors := content.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("数据验证失败: %v", errors)
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")

	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("type = ? AND DATE(date) = ?", learningType, todayStr).
		Delete(&models.LearningRecord{}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("删除旧记录失败: %v", err)
	}

	record := models.LearningRecord{
		Type:           learningType,
		Content:        content.Content,
		Interpretation: content.Interpretation,
		KeyWords:       content.FormatKeyWords(),
		Date:           today,
	}

	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("保存学习记录失败: %v", err)
	}

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