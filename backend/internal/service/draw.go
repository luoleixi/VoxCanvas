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
	if s.DB != nil {
		imageID, err := s.DB.CurrentImageID(sessionID)
		if err != nil {
			return nil, err
		}
		previousImageID = imageID
	} else if s.Generated != nil {
		if result, ok := s.Generated.Get(sessionID); ok {
			previousImageID = result.ImageID
		}
	}
	beforeDev := s.Dev.Get(sessionID)
	var sentenceID int64
	if s.DB != nil {
		var err error
		sentenceID, err = s.DB.RecordSentence(sessionID, previousImageID, sentence, "user_input", beforeDev)
		if err != nil {
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
		if s.DB != nil {
			title, summary := BuildSessionMeta(refined)
			if err := s.DB.RecordRequirementRefined(sessionID, title, summary, db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "requirement_refined",
				SentenceID:      sentenceID,
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				Dev:             refined,
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			}); err != nil {
				return nil, err
			}
		} else if err := s.setSessionDev(sessionID, refined); err != nil {
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
			title, summary := BuildSessionMeta(prompt)
			imageID, err = s.DB.RecordGeneratedImage(sessionID, prompt, base64Img, title, summary, db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "image_generated",
				SentenceID:      sentenceID,
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				Dev:             prompt,
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			})
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
		log.Printf("[DRAW] generate_image done session_id=%s image_id=%d image_len=%d", sessionID, imageID, len(base64Img))
		s.Dev.Set(sessionID, "")
		if s.DB == nil {
			if err := s.setSessionDev(sessionID, ""); err != nil {
				return nil, err
			}
		}
		return &model.DrawData{
			Op:    "generate_image",
			Text:  "",
			Image: base64Img,
		}, nil

	case "switch_session":
		targetSessionID := ""
		targetReason := "new_session"
		if s.Sessions != nil {
			target, err := s.Sessions.ResolveSwitchTarget(clientID, sessionID, sentence)
			if err != nil {
				return nil, err
			}
			if target != nil && target.Found {
				targetSessionID = target.Session.SessionID
				targetReason = target.Reason
				s.Dev.Set(targetSessionID, target.Session.Dev)
			}
		}
		if targetSessionID == "" {
			targetSessionID = NewSessionID()
		}
		if s.DB != nil {
			if err := s.DB.RecordSwitchSession(clientID, targetSessionID, db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "switch_session",
				SentenceID:      sentenceID,
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				Dev:             beforeDev,
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			}); err != nil {
				return nil, err
			}
		} else if s.Sessions != nil {
			if err := s.Sessions.Create(clientID, targetSessionID); err != nil {
				return nil, err
			}
		}
		log.Printf("[DRAW] switch_session client_id=%s from_session_id=%s to_session_id=%s reason=%s", clientID, sessionID, targetSessionID, targetReason)
		return &model.DrawData{
			Op:        "switch_session",
			Text:      "",
			Image:     "",
			SessionID: targetSessionID,
		}, nil

	case "list_sessions":
		text := "暂无历史会话。"
		sessionItems := make([]model.DrawSessionData, 0)
		if s.Sessions != nil {
			formatted, historySessions, err := s.Sessions.FormatHistoryForVoice(clientID, sessionID, 5)
			if err != nil {
				return nil, err
			}
			text = formatted
			for _, session := range historySessions {
				sessionItems = append(sessionItems, model.DrawSessionData{
					SessionID: session.SessionID,
					Title:     session.Title,
					Summary:   session.Summary,
				})
			}
		}
		if s.DB != nil {
			if err := s.DB.InsertSessionEvent(db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "list_sessions",
				SentenceID:      sentenceID,
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				Dev:             beforeDev,
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			}); err != nil {
				return nil, err
			}
		}
		log.Printf("[DRAW] list_sessions client_id=%s session_id=%s text_len=%d", clientID, sessionID, len(text))
		return &model.DrawData{
			Op:       "list_sessions",
			Text:     text,
			Image:    "",
			Sessions: sessionItems,
		}, nil

	case "undo":
		if s.DB != nil {
			image, err := s.DB.RecordUndoToPreviousImage(sessionID, db.SessionEvent{
				SessionID:       sessionID,
				EventType:       "undo",
				SentenceID:      sentenceID,
				PreviousImageID: previousImageID,
				Sentence:        sentence,
				BeforeDev:       beforeDev,
				BeforeImageID:   previousImageID,
			})
			if err != nil {
				return nil, err
			}
			if image == nil {
				log.Printf("[DRAW] undo miss session_id=%s reason=no_previous_image previous_image_id=%d", sessionID, previousImageID)
				return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
			}
			s.Dev.Set(sessionID, image.Prompt)
			if s.Generated != nil {
				s.Generated.Set(sessionID, GeneratedResult{
					ImageID: image.ImageID,
					Text:    image.Prompt,
					Image:   image.Base64Data,
				})
			}
			log.Printf("[DRAW] undo hit session_id=%s image_id=%d text_len=%d image_len=%d", sessionID, image.ImageID, len(image.Prompt), len(image.Base64Data))
			return &model.DrawData{
				Op:    "undo",
				Text:  image.Prompt,
				Image: image.Base64Data,
			}, nil
		}

		if s.Generated == nil {
			log.Printf("[DRAW] undo miss session_id=%s reason=no_generated_store", sessionID)
			return &model.DrawData{Op: "undo", Text: "", Image: ""}, nil
		}
		result, ok := s.Generated.UndoPrevious(sessionID)
		if !ok {
			log.Printf("[DRAW] undo miss session_id=%s reason=no_previous_generated_result", sessionID)
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
			if s.Generated != nil {
				s.Generated.Clear(sessionID)
			}
			if s.DB != nil {
				if err := s.DB.RecordClear(sessionID, db.SessionEvent{
					SessionID:       sessionID,
					EventType:       "clear",
					SentenceID:      sentenceID,
					PreviousImageID: previousImageID,
					Sentence:        sentence,
					Dev:             "",
					BeforeDev:       beforeDev,
					BeforeImageID:   previousImageID,
				}); err != nil {
					return nil, err
				}
			} else if err := s.setSessionDev(sessionID, ""); err != nil {
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
