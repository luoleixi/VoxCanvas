package llm

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const placeholderPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

type MockClassifier struct{}

func (m *MockClassifier) Classify(sentence string) (bool, string, error) {
	kw := []string{"生成", "画", "图片", "绘图", "绘制"}
	for _, k := range kw {
		if strings.Contains(sentence, k) {
			return true, sentence, nil
		}
	}
	return false, sentence, nil
}

type MockRefiner struct {
	counter int
}

func (m *MockRefiner) Refine(dev, newSentence string) (string, error) {
	m.counter++
	if dev == "" {
		return fmt.Sprintf("[需求 %d] %s", m.counter, newSentence), nil
	}
	return dev + "\n" + fmt.Sprintf("[需求 %d] %s", m.counter, newSentence), nil
}

type MockGenerator struct{}

func (m *MockGenerator) Generate(prompt string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(placeholderPNG)
	if err != nil {
		return "", err
	}
	return "base64:" + base64.StdEncoding.EncodeToString(b), nil
}
