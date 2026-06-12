package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type RealClient struct {
	Endpoint string
	APIKey   string
	Model    string
	client   *http.Client
}

func NewRealClient(endpoint, apiKey, model string) *RealClient {
	return &RealClient{
		Endpoint: strings.TrimRight(endpoint, "/"),
		APIKey:   apiKey,
		Model:    model,
		client:   &http.Client{},
	}
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

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type classifyResult struct {
	Op      string `json:"op"`
	Content string `json:"content"`
}

func (c *RealClient) Classify(sentence string) (bool, string, error) {
	sysPrompt := `你是一个语音绘图工具的指令分类器。用户通过语音输入文本，你需要判断文本是指令(order)还是需求(requirement)。

- order: 用户要求执行操作，如"生成图片"、"画一个圆"、"绘制"
- requirement: 用户描述想要的内容，如"红色的"、"一只猫"、"背景是蓝色"

请严格只输出一行JSON，不要用markdown代码块包裹：
{"op":"order", "content":"<原句或精简的指令>"}
或
{"op":"requirement", "content":"<原句>"}`

	raw, err := c.chat(sysPrompt, sentence)
	if err != nil {
		return false, "", err
	}

	cleaned := extractJSON(raw)
	var result classifyResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return false, "", fmt.Errorf("parse classify response: %w, raw: %s", err, raw)
	}

	isOrder := result.Op == "order"
	return isOrder, result.Content, nil
}

func (c *RealClient) Refine(dev, newSentence string) (string, error) {
	sysPrompt := `你是一个语音绘图工具的提示词精炼器。用户通过语音碎片化地描述想要的图像，你需要将这些碎片精炼成一段可用于AI生图的完整英文提示词。

规则：
1. 将新需求与已有上下文融合，写成一整段流畅的中文描述
2. 根据描述补充合理的视觉细节：光线、风格、构图、视角、色彩
3. 重复/矛盾的描述以最新为准，旧的直接覆盖
4. 只输出提示词文本，不要任何解释、引号或markdown格式`

	userMsg := fmt.Sprintf("当前已累积的提示词：\n%s\n\n用户最新语音输入：%s\n\n请将以上融合精炼为一段完整的中文生图提示词。", dev, newSentence)
	if dev == "" {
		userMsg = fmt.Sprintf("用户首次描述：%s\n\n请将其转化为一段完整的中文生图提示词。", newSentence)
	}

	return c.chat(sysPrompt, userMsg)
}

func (c *RealClient) chat(system, user string) (string, error) {
	reqBody := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.1,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := c.Endpoint + "/v1/chat/completions"
	log.Printf("[LLM] request -> %s model=%s user_len=%d", url, c.Model, len(user))

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[LLM] request error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LLM] read body error: %v", err)
		return "", err
	}

	log.Printf("[LLM] response <- status=%d body=%s", resp.StatusCode, truncate(string(body), 500))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deepseek api error %d: %s", resp.StatusCode, string(body))
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", err
	}

	if len(cr.Choices) == 0 {
		return "", errors.New("deepseek returned empty choices")
	}

	result := strings.TrimSpace(cr.Choices[0].Message.Content)
	log.Printf("[LLM] result: %s", truncate(result, 300))
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractJSON strips markdown code fences and extra text around JSON
func extractJSON(raw string) string {
	raw = strings.TrimSpace(raw)

	// strip ```json and ``` fences
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		end := strings.Index(raw, "```")
		if end != -1 {
			raw = raw[:end]
		}
		raw = strings.TrimSpace(raw)
	}

	// find JSON object boundaries: first { to last }
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		raw = raw[start : end+1]
	}

	return raw
}

// RealGenerator — DeepSeek 不支持生图，保留 stub
type RealGenerator struct{}

func (g *RealGenerator) Generate(prompt string) (string, error) {
	return "", errors.New("image generation requires a separate image API (e.g. DALL-E, Stable Diffusion)")
}
