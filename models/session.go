package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// Session модель для работы с сессиями
type Session struct {
	Token        string
	Info         *InfoUser
	ExpiresAfter time.Duration
}

// Validate валидация сессии
func (s *Session) Validate() *Errors {
	errs := Errors{
		Fields: make(map[string]*Error),
	}

	if s.Info == nil {
		errs.Fields["info"] = &Error{
			Code:        FailedToValidate,
			Description: "Session info is nil",
		}
	}

	if len(errs.Fields) == 0 {
		return nil
	}

	return &errs
}

// Set валидирует и сохраняет сессию в хранилище по сгенерированному токену
// Токен сохраняется в s.Token
func (s *Session) Set() *Errors {
	errs := s.Validate()
	if errs != nil {
		return errs
	}

	sessionToken, err := uuid.NewV4()
	if err != nil {
		return &Errors{
			Other: &Error{
				Code:        CantCreate,
				Description: fmt.Sprintf("unable to generate token: %s", err.Error()),
			},
		}
	}
	sessionInfo, err := json.Marshal(s.Info)
	if err != nil {
		return &Errors{
			Other: &Error{
				Code:        WrongJSON,
				Description: fmt.Sprintf("unable to marshal user info: %s", err.Error()),
			},
		}
	}

	_, err = storage.Do("SETEX", sessionToken.String(),
		int(s.ExpiresAfter.Seconds()), sessionInfo)
	if err != nil {
		return &Errors{
			Other: &Error{
				Code:        InternalStorage,
				Description: err.Error(),
			},
		}
	}

	s.Token = sessionToken.String()
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (s *Session) Delete() *Errors {
	_, err := storage.Do("DEL", s.Token)
	if err != nil {
		return &Errors{
			Other: &Error{
				Code:        InternalStorage,
				Description: err.Error(),
			},
		}
	}
	return nil
}

// GetSession получает сессию из хранилища по токену
func GetSession(token string) (*Session, *Errors) {
	data, err := redis.Bytes(storage.Do("GET", token))
	if err != nil {
		return nil, &Errors{
			Other: &Error{
				Code:        InternalStorage,
				Description: err.Error(),
			},
		}
	}
	userInfo := &InfoUser{}
	err = json.Unmarshal(data, userInfo)
	if err != nil {
		return nil, &Errors{
			Other: &Error{
				Code:        WrongJSON,
				Description: err.Error(),
			},
		}
	}
	return &Session{
		Token: token,
		Info:  userInfo,
	}, nil
}
