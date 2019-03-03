package models

import (
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Session модель для работы с сессиями
type Session struct {
	Token        string
	Payload      []byte
	ExpiresAfter time.Duration
}

// Set валидирует и сохраняет сессию в хранилище по сгенерированному токену
// Токен сохраняется в s.Token
func (s *Session) Set() error {
	sessionToken, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "session token generate error")
	}

	_, err = storage.Do("SETEX", sessionToken.String(),
		int(s.ExpiresAfter.Seconds()), s.Payload)
	if err != nil {
		return errors.Wrap(err, "redis save error")
	}

	s.Token = sessionToken.String()
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (s *Session) Delete() error {
	_, err := storage.Do("DEL", s.Token)
	if err != nil {
		return errors.Wrap(err, "redis delete error")
	}

	return nil
}

// GetSession получает сессию из хранилища по токену
func GetSession(token string) (*Session, error) {
	data, err := redis.Bytes(storage.Do("GET", token))
	if err != nil {
		return nil, errors.Wrap(err, "redis get error")
	}

	return &Session{
		Token:   token,
		Payload: data,
	}, nil
}
