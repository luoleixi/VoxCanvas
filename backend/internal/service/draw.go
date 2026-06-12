package service

import (
	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/model"
)

type DrawService struct {
	Dev       *DevStore
	Classifier llm.Classifier
	Refiner    llm.Refiner
	Generator  llm.Generator
	DB        *db.DB
}

func (s *DrawService) Handle(sentence string) (*model.DrawData, error) {
	if s.DB != nil {
		s.DB.InsertSentence(sentence, "user_input")
	}

	isOrder, _, err := s.Classifier.Classify(sentence)
	if err != nil {
		return nil, err
	}

	if isOrder {
		prompt := s.Dev.Get()
		base64Img, err := s.Generator.Generate(prompt)
		if err != nil {
			return nil, err
		}
		if s.DB != nil {
			s.DB.InsertImage(prompt, base64Img)
		}
		s.Dev.Set("")
		return &model.DrawData{
			Op:      "order",
			Content: base64Img,
		}, nil
	}

	refined, err := s.Dev.Append(sentence, s.Refiner)
	if err != nil {
		return nil, err
	}
	return &model.DrawData{
		Op:      "requirement",
		Content: refined,
	}, nil
}
