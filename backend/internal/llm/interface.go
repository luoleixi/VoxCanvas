package llm

type Classifier interface {
	Classify(sentence string) (isOrder bool, content string, err error)
}

type Refiner interface {
	Refine(dev, newSentence string) (refined string, err error)
}

type Generator interface {
	Generate(prompt string) (base64Image string, err error)
}
