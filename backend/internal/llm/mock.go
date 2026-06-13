package llm

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const placeholderPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

type MockClassifier struct{}

func (m *MockClassifier) Classify(sentence string) (*IntentResult, error) {
	switch {
	case containsAny(sentence, "撤销", "撤回", "回退", "恢复上一步", "不要刚才"):
		return &IntentResult{Op: "undo", Text: "", Image: ""}, nil
	case containsAny(sentence, "清空", "全部删掉", "全部删除", "从头开始", "重新画"):
		return &IntentResult{Op: "clear", Text: "", Image: ""}, nil
	case containsAny(sentence, "展示历史", "查看历史", "显示历史", "列出历史", "最近会话", "有哪些会话", "读一下历史", "播报历史") &&
		!containsAny(sentence, "打开", "回到", "切换", "切回", "返回"):
		return &IntentResult{Op: "list_sessions", Text: "", Image: ""}, nil
	case containsAny(sentence, "切换", "打开上一个", "回到", "历史会话", "历史记录"):
		return &IntentResult{Op: "switch_session", Text: "", Image: ""}, nil
	case containsAny(sentence, "生成图片", "生成图", "出图", "开始画", "帮我生成", "就按这个画"):
		return &IntentResult{Op: "generate_image", Text: "", Image: ""}, nil
	case strings.TrimSpace(sentence) == "":
		return &IntentResult{Op: "unknown", Text: "", Image: ""}, nil
	default:
		return &IntentResult{Op: "requirement", Text: sentence, Image: ""}, nil
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

type MockRefiner struct {
	counter int
}

func (m *MockRefiner) Refine(dev, newSentence string) (string, error) {
	m.counter++
	dev = strings.TrimPrefix(dev, "[mock] ")
	dev = strings.TrimSpace(dev)
	if dev == "" {
		return fmt.Sprintf("[mock] %s", newSentence), nil
	}
	return fmt.Sprintf("[mock] %s；%s", dev, newSentence), nil
}

type MockGenerator struct{}

func (m *MockGenerator) Generate(prompt string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(placeholderPNG)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
