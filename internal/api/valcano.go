package api

import (
	"bytes"
	"encoding/json"
	"everyday-study-backend/internal/config"
	"everyday-study-backend/internal/models"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type VolcanoClient struct {
	client  *http.Client
	config  *config.Config
}

func NewVolcanoClient(cfg *config.Config) *VolcanoClient {
	return &VolcanoClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: cfg,
	}
}

// 调用 Volcano API
func (vc *VolcanoClient) CallVolcanoAPI(learningType string, learned []string) (*models.VolcanoAPIResponse, error) {
	systemPrompt := vc.generatePrompt(learningType, learned)
	
	request := models.VolcanoAPIRequest{
		Model: "doubao-1.5-thinking-pro-250415",
		Messages: []models.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: "请给我推荐新的学习内容",
			},
		},
		Temperature: 0.7,
		MaxTokens:   1500,
		ResponseFormat: &models.ResponseFormat{
			Type: "json_object",
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", vc.config.VolcanoBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vc.config.VolcanoAPIKey)

	resp, err := vc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResponse models.VolcanoAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("API返回空响应")
	}

	return &apiResponse, nil
}

// 生成提示词
func (vc *VolcanoClient) generatePrompt(learningType string, learned []string) string {
	learnedText := strings.Join(learned, "\n")

	switch strings.ToLower(learningType) {
	case "english":
		return fmt.Sprintf(`你的任务是为一位想要学习英语谚语的人提供一句新的英语谚语，且不能与他已经学过的内容重复。
以下是他已经学过的英语谚语内容：
<learned_proverbs>
%s
</learned_proverbs>
在挑选新的英语谚语时，请确保它与已学内容不重复，句子来源可以是英语传统谚语、格言、习语等。
请以字符串json的格式输出内容，包含以下字段：
- proverb：英语谚语原文
- interpretation：谚语释义（包含中文翻译和含义解释）
- key_words：谚语中的关键词汇解析，格式为数组，每个元素包含word和meaning字段（排除简单的介词、冠词、代词等基础词汇，重点提取名词、动词、形容词等实义词）`, learnedText)

	case "chinese":
		return fmt.Sprintf(`你的任务是为一位想要学习中国传统诗词的人提供一句新的诗词，且不能与他已经学过的内容重复。
以下是他已经学过的诗词内容：
<learned_poems>
%s
</learned_poems>
在挑选新的诗词时，请确保它与已学内容不重复，句子来源可以是古诗、词、赋等中国传统文化中的诗词歌赋。
请以字符串json的格式输出内容，包含以下字段：
- poem：诗词原文
- interpretation：诗词释义和背景介绍
- key_words：诗词中的关键词汇解析，格式为数组，每个元素包含word和meaning字段`, learnedText)

	case "tcm":
		return fmt.Sprintf(`你的任务是为一位想要学习中医知识的人提供一条新的中医经典条文，且不能与他已经学过的内容重复。
以下是他已经学过的中医内容：
<learned_tcm>
%s
</learned_tcm>
在挑选新的中医条文时，请确保它与已学内容不重复，内容来源可以是《黄帝内经》、《伤寒论》、药性歌诀、汤头歌诀等中医经典。
请以字符串json的格式输出内容，包含以下字段：
- tcm_text：中医条文原文
- interpretation：条文释义和临床意义
- key_concepts：关键概念解析，格式为数组，每个元素包含concept和meaning字段`, learnedText)

	default:
		return fmt.Sprintf("不支持的学习类型: %s", learningType)
	}
}