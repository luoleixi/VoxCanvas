package llm

import (
	"bytes"
	"encoding/base64"
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

- order: 仅当用户**纯粹**表达"生成图片"意图，且句中**不带任何具体内容名词**（如物体、颜色、场景、风格等）时，才返回 order。例如"生成图片"、"画图"、"开始画"、"创建图片"、"帮我画一张"、"画出来吧"属于 order。content 固定为"生成图片"。
- requirement: 只要句中携带了具体绘制内容，无论是否包含"画"字，一律归为 requirement。例如"我想画一朵花"、"画只猫"、"红色的"、"蓝天白云"属于 requirement。

请严格只输出一行JSON，不要用markdown代码块包裹：
{"op":"order", "content":"生成图片"}
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

// RealGenerator — DashScope image generation
type RealGenerator struct {
	Endpoint string
	APIKey   string
	Model    string
	client   *http.Client
}

func NewRealGenerator(endpoint, apiKey, model string) *RealGenerator {
	return &RealGenerator{
		Endpoint: strings.TrimRight(endpoint, "/"),
		APIKey:   apiKey,
		Model:    model,
		client:   &http.Client{},
	}
}

type imageGenMessage struct {
	Role    string `json:"role"`
	Content []imageGenContent `json:"content"`
}

type imageGenContent struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type imageGenInput struct {
	Messages []imageGenMessage `json:"messages"`
}

type imageGenParams struct {
	NegativePrompt string `json:"negative_prompt"`
	PromptExtend   bool   `json:"prompt_extend"`
	Watermark      bool   `json:"watermark"`
	Size           string `json:"size"`
}

type imageGenRequest struct {
	Model      string         `json:"model"`
	Input      imageGenInput  `json:"input"`
	Parameters imageGenParams `json:"parameters"`
}

type imageGenResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content []imageGenContent `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
}

func (g *RealGenerator) Generate(prompt string) (string, error) {
	reqBody := imageGenRequest{
		Model: g.Model,
		Input: imageGenInput{
			Messages: []imageGenMessage{
				{
					Role: "user",
					Content: []imageGenContent{
						{Text: prompt},
					},
				},
			},
		},
		Parameters: imageGenParams{
			NegativePrompt: "低分辨率，低画质，肢体畸形，手指畸形，画面过饱和，蜡像感，人脸无细节，过度光滑，画面具有AI感。构图混乱。文字模糊，扭曲。",
			PromptExtend:   true,
			Watermark:      false,
			Size:           "1280*720",
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := g.Endpoint + "/api/v1/services/aigc/multimodal-generation/generation"
	log.Printf("[IMAGE] request -> %s model=%s prompt=%s", url, g.Model, truncate(prompt, 200))

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.APIKey)

	resp, err := g.client.Do(req)
	if err != nil {
		log.Printf("[IMAGE] request error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[IMAGE] response <- status=%d body=%s", resp.StatusCode, truncate(string(body), 500))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("dashscope api error %d: %s", resp.StatusCode, string(body))
	}

	var ir imageGenResponse
	if err := json.Unmarshal(body, &ir); err != nil {
		return "", err
	}

	if len(ir.Output.Choices) == 0 || len(ir.Output.Choices[0].Message.Content) == 0 {
		return "", errors.New("dashscope returned empty image")
	}

	img := ir.Output.Choices[0].Message.Content[0].Image
	if img == "" {
		return "", errors.New("dashscope returned empty image URL")
	}

	// if image is a URL, download it; if already base64, return directly
	if strings.HasPrefix(img, "data:") {
		return "base64:" + img, nil
	}
	if strings.HasPrefix(img, "http") {
		b64, err := downloadImage(g.client, img)
		if err != nil {
			return "", fmt.Errorf("download image: %w", err)
		}
		return "base64:" + b64, nil
	}

	return "base64:" + img, nil
}

func downloadImage(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
