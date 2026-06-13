package service

import (
	"log"

	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/model"
)

type DrawService struct {
	Dev        *DevStore
	Generated  *GeneratedStore
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
	var previousImageID int64
	if s.Generated != nil {
		if result, ok := s.Generated.Get(sessionID); ok {
			previousImageID = result.ImageID
		}
	}
	if s.DB != nil {
		s.DB.InsertSentence(sessionID, previousImageID, sentence, "user_input")
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
		if err := s.setSessionDev(sessionID, refined); err != nil {
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
			if err := s.setSessionDev(sessionID, ""); err != nil {
				return nil, err
			}
			return &model.DrawData{
				Op:    "generate_image",
				Text:  "",
				Image: "",
			}, nil
		}
		var imageID int64
		if s.DB != nil {
			imageID, err = s.DB.InsertImage(sessionID, prompt, base64Img)
			if err != nil {
				return nil, err
			}
		}
		if s.Generated != nil {
			s.Generated.Set(sessionID, GeneratedResult{
				ImageID: imageID,
				Text:    prompt,
				Image:   base64Img,
			})
		}
		s.Dev.Set(sessionID, "")
		if err := s.setSessionDev(sessionID, ""); err != nil {
			return nil, err
		}
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

	case "undo":
		if s.Generated == nil {
			return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
		}
		result, ok := s.Generated.Get(sessionID)
		if !ok {
			return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
		}
		s.Dev.Set(sessionID, result.Text)
		if err := s.setSessionDev(sessionID, result.Text); err != nil {
			return nil, err
		}
		return &model.DrawData{
			Op:    "undo",
			Text:  result.Text,
			Image: result.Image,
		}, nil

	case "clear", "unknown":
		if intent.Op == "clear" {
			s.Dev.Set(sessionID, "")
			if err := s.setSessionDev(sessionID, ""); err != nil {
				return nil, err
			}
			if s.Generated != nil {
				s.Generated.Clear(sessionID)
			}
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

func (s *DrawService) setSessionDev(sessionID, dev string) error {
	if s.Sessions == nil {
		return nil
	}
	return s.Sessions.SetDev(sessionID, dev)
}
