package models

import (
	"strings"
	"time"
)

type LearningRecord struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Type           string    `json:"type" gorm:"not null"`
	Content        string    `json:"content" gorm:"type:text;not null"`
	Interpretation string    `json:"interpretation" gorm:"type:text;not null"`
	KeyWords       string    `json:"key_words" gorm:"type:text"`
	Date           time.Time `json:"date" gorm:"type:date;not null"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LearnedContent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
}

type LearningType string

const (
	English LearningType = "english"
	Chinese LearningType = "chinese"
	TCM     LearningType = "tcm"
)

var LearningTypeNames = map[LearningType]string{
	English: "英语谚语",
	Chinese: "中文古诗词",
	TCM:     "中医基础",
}

func IsValidLearningType(t string) bool {
	switch LearningType(strings.ToLower(t)) {
	case English, Chinese, TCM:
		return true
	default:
		return false
	}
}

func GetAllLearningTypes() []string {
	return []string{"english", "chinese", "tcm"}
}

func GetLearningTypeName(t string) string {
	if name, ok := LearningTypeNames[LearningType(strings.ToLower(t))]; ok {
		return name
	}
	return t
}

func (lr *LearningRecord) FormatKeyWords() []string {
	if lr.KeyWords == "" {
		return []string{}
	}
	words := strings.Split(lr.KeyWords, ",")
	var result []string
	for _, word := range words {
		if trimmed := strings.TrimSpace(word); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
	Errors    []string    `json:"errors,omitempty"`
}

type TodayLearningData struct {
	Type           string    `json:"type"`
	TypeName       string    `json:"type_name"`
	Content        string    `json:"content"`
	Interpretation string    `json:"interpretation"`
	KeyWords       []string  `json:"key_words"`
	Date           string    `json:"date"`
	FromCache      bool      `json:"from_cache"`
}

type LearningHistoryData struct {
	Total   int                    `json:"total"`
	Records []LearningHistoryItem  `json:"records"`
}

type LearningHistoryItem struct {
	Type           string   `json:"type"`
	TypeName       string   `json:"type_name"`
	Content        string   `json:"content"`
	Interpretation string   `json:"interpretation"`
	KeyWords       []string `json:"key_words"`
	Date           string   `json:"date"`
}

type UserStatsData struct {
	Stats map[string]TypeStats `json:"stats"`
}

type TypeStats struct {
	TypeName   string `json:"type_name"`
	TotalDays  int    `json:"total_days"`
	UniqueDays int    `json:"unique_days"`
}

type HealthData struct {
	Status         string   `json:"status"`
	Database       string   `json:"database"`
	SupportedTypes []string `json:"supported_types"`
}

type VolcanoAPIRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type VolcanoAPIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type AIContent struct {
	Proverb string `json:"proverb,omitempty"`
	Poem string `json:"poem,omitempty"`
	TCMText string `json:"tcm_text,omitempty"`
	Interpretation string `json:"interpretation"`
	KeyWords []KeyWordItem `json:"key_words,omitempty"`
	KeyConcepts []KeyConceptItem `json:"key_concepts,omitempty"`
}

type KeyWordItem struct {
	Word    string `json:"word"`
	Meaning string `json:"meaning"`
}

type KeyConceptItem struct {
	Concept string `json:"concept"`
	Meaning string `json:"meaning"`
}

type LearningContent struct {
	Type           LearningType
	Content        string
	Interpretation string
	KeyWords       []string
	Date           time.Time
}

func (lc *LearningContent) Validate() []string {
	var errors []string
	
	if strings.TrimSpace(lc.Content) == "" {
		errors = append(errors, "内容不能为空")
	}
	if strings.TrimSpace(lc.Interpretation) == "" {
		errors = append(errors, "解释不能为空")
	}
	if len(lc.KeyWords) == 0 {
		errors = append(errors, "关键词不能为空")
	}
	
	return errors
}

func (lc *LearningContent) FormatKeyWords() string {
	return strings.Join(lc.KeyWords, ",")
}