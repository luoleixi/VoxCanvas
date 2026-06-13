package llm

type IntentResult struct {
	Op    string `json:"op"`
	Text  string `json:"text"`
	Image string `json:"image"`
}

type Classifier interface {
	Classify(sentence string) (*IntentResult, error)
}

type Refiner interface {
	Refine(dev, newSentence string) (refined string, err error)
}

type Generator interface {
	Generate(prompt string) (base64Image string, err error)
}
