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
	beforeDev := s.Dev.Get(sessionID)
	if s.DB != nil {
		s.DB.InsertSentence(sessionID, previousImageID, sentence, "user_input")
		if err := s.insertEvent(db.SessionEvent{
			SessionID:       sessionID,
			EventType:       "sentence",
			PreviousImageID: previousImageID,
			Sentence:        sentence,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		}); err != nil {
			return nil, err
		}
	}

	intent, err := s.Classifier.Classify(sentence)
	if err != nil {
		return nil, err
	}
	log.Printf("[DRAW] intent client_id=%s session_id=%s op=%s", clientID, sessionID, intent.Op)

	switch intent.Op {
	case "requirement":
		refined, err := s.Dev.Append(sessionID, sentence, s.Refiner)
		if err != nil {
			return nil, err
		}
		if err := s.setSessionDev(sessionID, refined); err != nil {
			return nil, err
		}
		if err := s.insertEvent(db.SessionEvent{
			SessionID:       sessionID,
			EventType:       "requirement_refined",
			PreviousImageID: previousImageID,
			Sentence:        sentence,
			Dev:             refined,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		}); err != nil {
			return nil, err
		}
		log.Printf("[DRAW] requirement_refined session_id=%s text_len=%d previous_image_id=%d", sessionID, len(refined), previousImageID)
		return &model.DrawData{
			Op:    "requirement",
			Text:  refined,
			Image: "",
		}, nil

	case "generate_image":
		prompt := s.Dev.Get(sessionID)
		log.Printf("[DRAW] generate_image start session_id=%s prompt_len=%d previous_image_id=%d", sessionID, len(prompt), previousImageID)
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
		if err := s.insertEvent(db.SessionEvent{
			SessionID:       sessionID,
			EventType:       "image_generated",
			ImageID:         imageID,
			PreviousImageID: previousImageID,
			Sentence:        sentence,
			Dev:             prompt,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		}); err != nil {
			return nil, err
		}
		log.Printf("[DRAW] generate_image done session_id=%s image_id=%d image_len=%d", sessionID, imageID, len(base64Img))
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
		if err := s.insertEvent(db.SessionEvent{
			SessionID:       sessionID,
			EventType:       "switch_session",
			PreviousImageID: previousImageID,
			Sentence:        sentence,
			Dev:             beforeDev,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		}); err != nil {
			return nil, err
		}
		log.Printf("[DRAW] switch_session client_id=%s from_session_id=%s to_session_id=%s", clientID, sessionID, newSessionID)
		return &model.DrawData{
			Op:        "switch_session",
			Text:      "",
			Image:     "",
			SessionID: newSessionID,
		}, nil

	case "undo":
		if s.Generated == nil {
			log.Printf("[DRAW] undo miss session_id=%s reason=no_generated_store", sessionID)
			return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
		}
		result, ok := s.Generated.Get(sessionID)
		if !ok {
			log.Printf("[DRAW] undo miss session_id=%s reason=no_generated_result", sessionID)
			return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
		}
		s.Dev.Set(sessionID, result.Text)
		if err := s.setSessionDev(sessionID, result.Text); err != nil {
			return nil, err
		}
		if err := s.insertEvent(db.SessionEvent{
			SessionID:       sessionID,
			EventType:       "undo",
			ImageID:         result.ImageID,
			PreviousImageID: previousImageID,
			Sentence:        sentence,
			Dev:             result.Text,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		}); err != nil {
			return nil, err
		}
		log.Printf("[DRAW] undo hit session_id=%s image_id=%d text_len=%d image_len=%d", sessionID, result.ImageID, len(result.Text), len(result.Image))
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
			if err := s.insertEvent(db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "clear",
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				Dev:             "",
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			}); err != nil {
				return nil, err
			}
			log.Printf("[DRAW] clear session_id=%s before_dev_len=%d before_image_id=%d", sessionID, len(beforeDev), previousImageID)
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

func (s *DrawService) insertEvent(event db.SessionEvent) error {
	if s.DB == nil {
		return nil
	}
	return s.DB.InsertSessionEvent(event)
}
