package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/getjobs/server/internal/manager"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/repository"
)

type AiService struct {
	aiRepo  *repository.AiRepo
	cfgRepo *repository.ConfigRepo
}

func NewAiService(aiRepo *repository.AiRepo, cfgRepo *repository.ConfigRepo) *AiService {
	return &AiService{aiRepo: aiRepo, cfgRepo: cfgRepo}
}

type aiApiConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

func (s *AiService) getAiApiConfig() (*aiApiConfig, error) {
	baseURL, _ := s.getConfigValue("BASE_URL")
	apiKey, _ := s.getConfigValue("API_KEY")
	model, _ := s.getConfigValue("MODEL")

	if baseURL == "" || apiKey == "" || model == "" {
		return nil, fmt.Errorf("AI API configuration incomplete: BASE_URL, API_KEY, MODEL must be set in config table")
	}

	return &aiApiConfig{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
	}, nil
}

func (s *AiService) getConfigValue(key string) (string, error) {
	cfg, err := s.cfgRepo.GetByKey(key)
	if err != nil {
		return "", err
	}
	return cfg.ConfigValue, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
}

func (s *AiService) Chat(content string) (string, error) {
	cfg, err := s.getAiApiConfig()
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		if strings.HasSuffix(endpoint, "/v1") || strings.Contains(endpoint, "/v1/") {
			endpoint += "/chat/completions"
		} else {
			endpoint += "/v1/chat/completions"
		}
	}

	reqBody := chatRequest{
		Model:       cfg.Model,
		Messages:    []chatMessage{{Role: "user", Content: content}},
		Temperature: 0.5,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("AI request failed: status=%d, body=%s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("AI request failed with status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	return string(respBody), nil
}

func (s *AiService) GetConfig() (interface{}, error) {
	return s.aiRepo.Get()
}

func (s *AiService) SaveConfig(introduce, prompt string) (*model.AiConfig, error) {
	cfg := &model.AiConfig{
		Introduce: introduce,
		Prompt:    prompt,
	}
	if err := s.aiRepo.Save(cfg); err != nil {
		return nil, err
	}
	return s.aiRepo.Get()
}

func (s *AiService) GenerateFromBoss() (*model.AiConfig, error) {
	resumeText, err := s.fetchBossResumeText()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Boss resume: %w", err)
	}

	prompt := s.buildAiConfigGenerationPrompt(resumeText)
	raw, err := s.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	parsed, err := s.extractJson(raw)
	if err != nil {
		return nil, err
	}

	introduce := ""
	promptResult := ""
	if v, ok := parsed["introduce"]; ok {
		introduce = v
	}
	if v, ok := parsed["prompt"]; ok {
		promptResult = v
	}

	if introduce == "" || promptResult == "" {
		return nil, fmt.Errorf("AI returned incomplete configuration")
	}

	cfg := &model.AiConfig{
		Introduce: introduce,
		Prompt:    promptResult,
	}
	if err := s.aiRepo.Save(cfg); err != nil {
		return nil, err
	}
	return s.aiRepo.Get()
}

func (s *AiService) fetchBossResumeText() (string, error) {
	bm := manager.DefaultManager
	page := bm.GetPage("boss")
	if page == nil {
		return "", fmt.Errorf("请先登录Boss直聘")
	}

	if err := page.Navigate("https://www.zhipin.com/web/geek/resume"); err != nil {
		return "", fmt.Errorf("failed to navigate to resume page: %w", err)
	}
	time.Sleep(3 * time.Second)

	el, err := page.Element("body")
	if err != nil {
		return "", fmt.Errorf("failed to get page body: %w", err)
	}

	text, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("failed to get page text: %w", err)
	}

	if len(text) > 3000 {
		text = text[:3000]
	}

	return text, nil
}

func (s *AiService) buildAiConfigGenerationPrompt(resumeText string) string {
	return fmt.Sprintf(`你是求职自动投递系统的配置生成助手。请根据当前Boss账号在线简历和求职目标，生成AI配置页面中的两项内容：introduce 和 prompt。

重要要求：
1. 当前候选人不一定是程序员，不要默认写Java、Python、技术栈、开发经验。必须严格依据简历内容判断候选人的实际职业方向。
2. introduce 是给AI使用的候选人画像，需包含：求职方向、目标岗位、期望城市/薪资、工作年限、核心经验、优势、可沟通亮点、需要规避或不要夸大的点。
3. prompt 是自动投递时生成HR打招呼语的模板，必须兼容 Go fmt.Sprintf 的5个占位符，且只能包含5个 %%s，占位符含义依次为：候选人介绍、搜索关键词、岗位名称、岗位JD、默认招呼语。
4. prompt 必须要求AI输出"只返回一句可直接发给HR的中文招呼语"，不要解释，不要Markdown，不要多余标点；如果岗位JD为空，也要基于候选人介绍、搜索关键词和岗位名称生成通用招呼语。
5. prompt 要强调基本符合才表达意向，不编造经历，不出现程序员/技术岗假设。
6. 输出严格JSON，不要Markdown代码块，不要JSON以外说明。

返回结构：
{
  "introduce": "适合粘贴到技能介绍框的完整文本",
  "prompt": "适合粘贴到AI提示词框的模板，必须且只包含5个%%s"
}

Boss在线简历和求职目标：
%s`, resumeText)
}

func (s *AiService) extractJson(raw string) (map[string]string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("AI returned empty content")
	}

	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in AI response")
	}
	text = text[start : end+1]

	var result map[string]string
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}
	return result, nil
}
