package service

import (
	"log"

	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/model"
)

type DrawService struct {
	Dev        *DevStore
	Classifier llm.Classifier
	Refiner    llm.Refiner
	Generator  llm.Generator
	DB         *db.DB
}

func (s *DrawService) Handle(sentence string) (*model.DrawData, error) {
	if s.DB != nil {
		s.DB.InsertSentence(sentence, "user_input")
	}

	intent, err := s.Classifier.Classify(sentence)
	if err != nil {
		return nil, err
	}

	switch intent.Op {
	case "requirement":
		refined, err := s.Dev.Append(sentence, s.Refiner)
		if err != nil {
			return nil, err
		}
		return &model.DrawData{
			Op:    "requirement",
			Text:  refined,
			Image: "",
		}, nil

	case "generate_image":
		prompt := s.Dev.Get()
		base64Img, err := s.Generator.Generate(prompt)
		if err != nil {
			log.Printf("[DRAW] image gen skipped: %v, return prompt as content", err)
			s.Dev.Set("")
			return &model.DrawData{
				Op:    "generate_image",
				Text:  "",
				Image: "",
			}, nil
		}
		if s.DB != nil {
			s.DB.InsertImage(prompt, base64Img)
		}
		s.Dev.Set("")
		return &model.DrawData{
			Op:    "generate_image",
			Text:  "",
			Image: base64Img,
		}, nil

	case "undo", "clear", "switch_session", "unknown":
		if intent.Op == "clear" {
			s.Dev.Set("")
		}
		return &model.DrawData{
			Op:    intent.Op,
			Text:  "",
			Image: "",
		}, nil

	default:
		return &model.DrawData{
			Op:    "unknown",
			Text:  "",
			Image: "",
		}, nil
	}
}
