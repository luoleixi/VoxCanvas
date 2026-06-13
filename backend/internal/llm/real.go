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

func (c *RealClient) Classify(sentence string) (*IntentResult, error) {
	sysPrompt := `你是 VoxCanvas 的语音绘图指令识别器。

用户只能通过语音控制绘图。你的任务是判断用户输入属于哪一种操作，并返回严格 JSON。

可用 op：
- requirement：用户描述、修改、补充绘图需求，例如画一只猫、加上月亮、背景换成森林、风格改成水彩、不要那只鸟。
- generate_image：用户要求生成图片、开始绘制、出图，例如生成图片、开始画吧、出图、帮我生成、就按这个画。
- undo：用户要求撤销上一步、回退、恢复上一步，例如撤销、撤回刚才那步、回退一步、不要刚才的修改。
- clear：用户要求清空当前画布或当前会话内容，例如清空、清空画布、全部删掉、从头开始、重新画。
- switch_session：用户要求切换、打开、返回某个历史会话，例如切换会话、打开上一个会话、回到刚才那个作品、切换到海边小屋那张。
- unknown：无法识别，或用户输入与绘图控制无关。

返回格式固定为：
{"op":"requirement|generate_image|undo|clear|switch_session|unknown","text":"","image":""}

字段规则：
1. op 必须是上面的枚举值之一。
2. text 只在 op=requirement 时填写用户的绘图需求文本，可以保留原句或轻微规范化；其他 op 必须返回空字符串。
3. image 永远返回空字符串，不能编造图片信息。

判断规则：
1. 描述画面内容、风格、颜色、构图、添加或删除元素，返回 requirement。
2. 明确要求生成图片、出图、开始画，返回 generate_image。
3. 撤销、回退、不要刚才那步，返回 undo。
4. 清空、全部删除、重新开始当前作品，返回 clear。
5. 切换、打开、回到某个历史会话，返回 switch_session。
6. 如果一句话同时包含切换会话和其他绘图需求，以 switch_session 为主。
7. 如果一句话同时包含绘图需求和生成图片，以 generate_image 为主，text 仍返回空字符串。
8. 不确定时返回 unknown。

请严格只输出一行 JSON，不要使用 markdown 代码块，不要解释。`

	raw, err := c.chat(sysPrompt, sentence)
	if err != nil {
		return nil, err
	}

	cleaned := extractJSON(raw)
	var result IntentResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("parse classify response: %w, raw: %s", err, raw)
	}

	return &result, nil
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
		return "", fmt.Errorf("dashscope chat api error %d: %s", resp.StatusCode, string(body))
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", err
	}

	if len(cr.Choices) == 0 {
		return "", errors.New("dashscope chat api returned empty choices")
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
	Role    string            `json:"role"`
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
		return img, nil
	}
	if strings.HasPrefix(img, "http") {
		b64, err := downloadImage(g.client, img)
		if err != nil {
			return "", fmt.Errorf("download image: %w", err)
		}
		return b64, nil
	}

	return img, nil
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
