package service

import (
	"log"

	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/model"
)

type DrawService struct {
	Dev        *DevStore
	Sessions   *SessionService
	Classifier llm.Classifier
	Refiner    llm.Refiner
	Generator  llm.Generator
	DB         *db.DB
}

func (s *DrawService) Handle(clientID, sessionID, sentence string) (*model.DrawData, error) {
	if s.Sessions != nil {
		if err := s.Sessions.Touch(clientID, sessionID); err != nil {
			return nil, err
		}
	}
	if s.DB != nil {
		s.DB.InsertSentence(sessionID, sentence, "user_input")
	}

	intent, err := s.Classifier.Classify(sentence)
	if err != nil {
		return nil, err
	}

	switch intent.Op {
	case "requirement":
		refined, err := s.Dev.Append(sessionID, sentence, s.Refiner)
		if err != nil {
			return nil, err
		}
		return &model.DrawData{
			Op:    "requirement",
			Text:  refined,
			Image: "",
		}, nil

	case "generate_image":
		prompt := s.Dev.Get(sessionID)
		base64Img, err := s.Generator.Generate(prompt)
		if err != nil {
			log.Printf("[DRAW] image gen skipped: %v, return prompt as content", err)
			s.Dev.Set(sessionID, "")
			return &model.DrawData{
				Op:    "generate_image",
				Text:  "",
				Image: "",
			}, nil
		}
		if s.DB != nil {
			s.DB.InsertImage(sessionID, prompt, base64Img)
		}
		s.Dev.Set(sessionID, "")
		return &model.DrawData{
			Op:    "generate_image",
			Text:  "",
			Image: base64Img,
		}, nil

	case "switch_session":
		newSessionID := NewSessionID()
		if s.Sessions != nil {
			if err := s.Sessions.Create(clientID, newSessionID); err != nil {
				return nil, err
			}
		}
		return &model.DrawData{
			Op:        "switch_session",
			Text:      "",
			Image:     "",
			SessionID: newSessionID,
		}, nil

	case "undo", "clear", "unknown":
		if intent.Op == "clear" {
			s.Dev.Set(sessionID, "")
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
